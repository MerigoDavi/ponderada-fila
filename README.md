# Ponderada Fila - Backend de Telemetria IoT

Este repositório contém a entrega da atividade, implementada em Go. A solução foca em lidar com o problema de gargalo ao receber requisições pesadas usando arquitetura orientada a eventos. Os serviços estão 100% conteinerizados usando Docker Compose.

---

## Decisões Tomadas para a Construção da Solução

Para desenvolver a aplicação como exigida pelo enunciado e respeitando as restrições teóricas do problema de IoT industriais, adotei as seguintes estratégias:

### 1. Separação de Responsabilidades
Ao invés de ter um único programa recebendo requisições HTTP e salvando diretamente no PostgreSQL, eu decidi dividir o projeto em dois serviços Go diferentes e independentes:
- **Backend API:** Um programa que expõe a rota POST via HTTP. Ele apenas pega o corpo JSON e envia como mensagem para o *RabbitMQ*.
- **Consumer:** Um programa apenas em escutar o RabbitMQ no background, desenfileirar os pacotes recebidos, formatar e dar *Insert* no banco de dados.

**Por quê?** Salvar no banco exige verificação no disco rígido e tempo. Se fosse feito na Rota Web, a conexão do dispositivo ficaria presa esperando a gravação completar. Agora ele só grava a mensagem na memória do RabbitMQ e libera o sensor industrial (`Status HTTP 202 - Accepted`). 

### 2. Uso do RabbitMQ 
O problema cita claramente surtos repentinos e picos de envios por alta concorrência. 
O *RabbitMQ* foi escolhido para o papel central de "Mensageria". Ele absorve esse choque. Se tiver mil dispositivos publicando dados ao mesmo tempo, a Fila do RabbitMQ armazena com folga.

### 3. PostgreSQL Simples com Tabela Eficiente
Mantive a tabela do Postgres enxuta em `db/init.sql`. Como no cenário IoT às vezes há variação entre se a métrica é um *número flutuante* (como temperatura 34.5) ou *discreta* (0 ligado, 1 desligado), escolhi centralizar o valor na tabela no formato "FLOAT" para uniformidade, mas acompanhado da coluna de texto classificatória "nature" (analog ou discrete) para a leitura não ficar confusa.

### 4. Golang
Os requisitos não amarram que deveria usar o GoLang, mas eu escolhi pela sua eficiência no tratamento de concorrência com as *Goroutines*, além de compilar binários pequenos nativos pro Docker e ter uma latência padrão pequena ao usar bibliotecas HTTP como o framework Web `Gin`.

---

## Como Executar a Aplicação

Toda a infraestrutura é montada através do Docker de forma autônoma.

### Para Inicializar Tudo
Para dar o "start" do banco, do broker e acionar os serviços em Go, digite no terminal (pasta raiz onde fica o `compose.yaml`):

```bash
docker-compose up -d --build
```
*(ou apenas `docker compose up -d --build` dependendo de como instalou seu plugin do docker)*

### Verificando os Logs do Processamento
Os containers da frente (backend) e fundo (consumer) rodam no host do docker. A porta é a 8080 caso queira mandar curls manualmente, mas caso queira ver o log de fato das mensagens entrando:

```bash
docker-compose logs -f consumer
```

---

## Teste de Carga e Comportamento Prático

Segundo os requisitos da atividade, precisávamos fazer valer via k6 que a requisição de fato segura o estresse.

### Como reporduzir o teste:
Usei um script do `k6` que está na pasta `/load-test`. 

Para iniciar o k6 sem precisar baixar ele, pode rodar o container dele de forma conectada na sua rede do terminal:
```bash
docker run --rm -i -e K6_NO_COLOR=1 --net host grafana/k6 run - < load-test/k6-script.js > load-test/saida-do-k6.txt
```

### Análise dos Dados Obtidos 
Durante meu teste de demonstração na fase de construção, o k6 subiu até quase **500 Dispositivos simultâneos** em um pico em loop de 60 segundos total.

* A métrica `http_req_duration` marcou um *P95* (Percentil 95%) de **~1.9 milisegundos**.
* A métrica `checks_succeeded` apontou que **todas as +112 mil requisições HTTP retornaram sucesso**.
* O *Throughput* de capacidade manteve-se quase na faixa dos **1.8 mil inserts/s** para a porta de entrada.

(Os resultados detalhados desse relatório analítico supracitado podem ser verificados lendo o arquivo completo gerado em [**`load-test/report.md`**](load-test/report.md)).
 
**Gargalos Encontrados e Futuros Limitadores:**
O teste relata puramente o quão rápido o sistema engole requisições. Contudo, em uma carga produtivamente ainda maior que perdure 24 horas ininterruptamente, o Consumer pode demorar até acabar de depositar a fila inteira para o Banco PostgreSQL — pois sua estrutura de inserção ali foi concebida de "1 a 1".

A melhoria explícita em um cenário real da empresa seria realizar atualizações de **Batch Inserts**. Ou seja, reter 200 dados em um buffer, e o *Consumer* realizar somente 1 mega-insert no SGDB ao invés de centenas isolados, o que pouparia tempo de rede. Para o escopo da capacidade da Fila no problema, atende a forma atual.

---

## Testes Unitários

Pensando em avaliar a coesão de conversão de dados JSON da aplicação isoladamente, foram implementadas Testes Unitários Nativos para ambos os microsserviços usando o pacote `testing` e `httptest` do Go:

- No **Backend**, o mock simula requisições POST para validar o formato de `Payload` e checar recusa em formatos malformados.
- No **Consumer**, se testa a lógica do `Unmarshal` para atestar a ingestão de campos antes de baterem no PostgreSQL.

**Para rodar as suítes (precisa ter o Go instalado localmente na máquina caso rode fora do Docker):**
```bash
# Rodar os testes do recebimento / router HTTP (Backend)
cd back
go test -v ./...

# Rodar testes de tratamento de fila (Consumer)
cd ../consumer
go test -v ./...
```
