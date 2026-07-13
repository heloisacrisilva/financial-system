# Financial System API

API REST em Go para processamento de transacoes financeiras com contas, clientes, historico, idempotencia por `reference_id`, controle de concorrencia por lock pessimista e eventos assincronos via outbox.

## Stack

- Go 1.25
- Gin para HTTP
- GORM para persistencia
- PostgreSQL
- Docker Compose para subir o banco local
- OpenAPI/Swagger para documentacao da API

## Como Rodar

### 1. Subir o PostgreSQL

```bash
docker-compose up -d
```

O banco sobe em `localhost:5435`.

### 2. Configurar variaveis de ambiente

O projeto espera:

```env
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=api_financial_system
DB_PORT=5435
DB_HOST=localhost
APP_PORT=8080
```

O arquivo `.env` local ja contem a configuracao do banco. `APP_PORT` e opcional; se nao informado, a API usa `8080`.

### 3. Executar a aplicacao

```bash
go run main.go
```

A API ficara disponivel em:

```text
http://localhost:8080
```

## Testes

```bash
go test ./...
```

Neste momento o projeto compila todos os pacotes, mas ainda nao possui arquivos `*_test.go`. A proxima evolucao recomendada e adicionar testes unitarios para regras de operacao e testes de integracao com PostgreSQL.

## Swagger / OpenAPI

Com a aplicacao rodando:

- Swagger UI: http://localhost:8080/swagger
- OpenAPI YAML: http://localhost:8080/openapi.yaml

O contrato tambem esta versionado em:

```text
api/docs/openapi.yaml
```

## Endpoints

### Health Check

```http
GET /health
```

### Contas

```http
POST /api/accounts/create
GET  /api/accounts/{id}
```

### Clientes

```http
GET /api/clients/{id}
```

### Operacoes

```http
POST /api/operations/{type}
GET  /api/operations/{id}
```

Tipos aceitos em `{type}`:

- `credit`
- `debit`
- `reserve`
- `capture`
- `transfer`
- `reversal`

## Exemplos de Uso

### Health Check

```bash
curl http://localhost:8080/health
```

Resposta esperada:

```json
{
  "status": "ok",
  "timestamp": "2026-07-13T10:00:00Z",
  "checks": {
    "api": "ok",
    "database": "ok"
  }
}
```

### Criar Conta

```bash
curl -X POST http://localhost:8080/api/accounts/create \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Maria Silva",
    "email": "maria@example.com",
    "cpf": "12345678901"
  }'
```

### Creditar Saldo

```bash
curl -X POST http://localhost:8080/api/operations/credit \
  -H "Content-Type: application/json" \
  -d '{
    "account_id": 1,
    "value": 100000,
    "currency": "BRL",
    "reference_id": "TXN-001"
  }'
```

### Debitar Saldo

```bash
curl -X POST http://localhost:8080/api/operations/debit \
  -H "Content-Type: application/json" \
  -d '{
    "account_id": 1,
    "value": 20000,
    "currency": "BRL",
    "reference_id": "TXN-002"
  }'
```

### Reservar Saldo

```bash
curl -X POST http://localhost:8080/api/operations/reserve \
  -H "Content-Type: application/json" \
  -d '{
    "account_id": 1,
    "value": 30000,
    "currency": "BRL",
    "reference_id": "TXN-003"
  }'
```

### Capturar Reserva

```bash
curl -X POST http://localhost:8080/api/operations/capture \
  -H "Content-Type: application/json" \
  -d '{
    "account_id": 1,
    "value": 30000,
    "currency": "BRL",
    "reference_id": "TXN-004"
  }'
```

### Transferir Entre Contas

```bash
curl -X POST http://localhost:8080/api/operations/transfer \
  -H "Content-Type: application/json" \
  -d '{
    "account_id": 1,
    "account_dest_id": 2,
    "value": 50000,
    "currency": "BRL",
    "reference_id": "TXN-005"
  }'
```

### Estornar Operacao

```bash
curl -X POST http://localhost:8080/api/operations/reversal \
  -H "Content-Type: application/json" \
  -d '{
    "original_reference_id": "TXN-005",
    "reference_id": "TXN-006"
  }'
```

### Consultar Historico Por ID

```bash
curl http://localhost:8080/api/operations/1
```

## Regras de Negocio Implementadas

- Credito adiciona valor ao saldo disponivel.
- Debito remove valor do saldo disponivel e considera limite de credito.
- Reserva move valor do saldo disponivel para saldo reservado.
- Captura confirma uma reserva conforme regra do projeto, retirando do reservado e refletindo no saldo disponivel.
- Transferencia movimenta valor entre duas contas.
- Estorno reverte creditos, debitos e transferencias.
- Contas precisam estar ativas para operar.
- Moeda da operacao deve bater com a moeda da conta quando informada.
- `reference_id` e idempotente no nivel da operacao.
- Operacoes financeiras sao atomicas com transacao de banco.
- Operacoes na mesma conta usam lock pessimista (`SELECT FOR UPDATE`) para consistencia em concorrencia.

## Idempotencia

A idempotencia e controlada pela tabela `operation_references`, que possui `reference_id` unico.

Esse controle fica separado de `transaction_histories` porque uma unica operacao pode gerar mais de uma linha de historico. Exemplo: uma transferencia gera uma perna de debito e outra de credito, ambas com o mesmo `reference_id`.

## Eventos Assincronos

O projeto usa o padrao outbox:

1. A operacao financeira grava saldo/historico dentro de uma transacao.
2. Na mesma transacao, grava um registro em `operation_events` com status `pending`.
3. Um worker em background busca eventos pendentes.
4. O worker publica o evento e marca como `published`.
5. Se falhar, marca como `failed` e agenda retry com backoff exponencial.

Hoje a publicacao real e simulada por log no metodo `publishEvent`. A estrutura esta preparada para substituir essa funcao por Kafka, RabbitMQ, SQS ou outro broker.

## Decisoes Tecnicas

- **Gin**: framework HTTP simples e direto para APIs REST.
- **GORM**: reduz boilerplate de persistencia e suporta transacoes, locks e migracao automatica.
- **PostgreSQL**: banco relacional adequado para consistencia transacional e controle de concorrencia.
- **Outbox pattern**: evita perder eventos quando a operacao de negocio foi persistida, mas a publicacao externa falha.
- **Lock pessimista**: usado nas contas durante operacoes financeiras para evitar race condition de saldo.
- **Retry com backoff**: aplicado tanto no handler para falhas retryable quanto no dispatcher de eventos.

## Estrutura do Projeto

```text
api/
  config/        Configuracao de banco e logger
  docs/          Contrato OpenAPI
  entities/      Modelos persistidos
  events/        Dispatcher em background dos eventos
  handler/       Handlers HTTP
  helpers/       Funcoes auxiliares
  repository/    Regras de persistencia e operacoes financeiras
  router/        Rotas HTTP
main.go          Bootstrap da aplicacao
docker-compose.yml
```

## Observabilidade

O projeto possui logs para:

- inicializacao da aplicacao;
- erros de validacao;
- erros de banco;
- falhas de operacoes;
- publicacao de eventos;
- falhas e retries no dispatcher de eventos.
- health check da API e conectividade com PostgreSQL.

## Limitacoes Conhecidas

- Ainda nao ha testes unitarios e de integracao automatizados.
