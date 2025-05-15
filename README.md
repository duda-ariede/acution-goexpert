# Projeto Auction GoExpert

Este projeto é um sistema de leilão implementado em Go, utilizando MongoDB como banco de dados e Gin como framework web. Ele fornece APIs RESTful para gerenciar leilões, lances e usuários.

## Pré-requisitos

Antes de executar o projeto, certifique-se de ter o seguinte instalado:

- [Go](https://golang.org/dl/) (versão 1.23 ou superior)
- [Docker](https://www.docker.com/get-started)
- [Docker Compose](https://docs.docker.com/compose/install/)

## Variáveis de Ambiente

O projeto requer que as seguintes variáveis de ambiente estejam definidas. Normalmente, elas são definidas no arquivo `cmd/auction/.env`:

- `MONGODB_URL`: URI de conexão com o MongoDB (exemplo: `mongodb://localhost:27017`)
- `MONGODB_DB`: Nome do banco de dados MongoDB a ser utilizado

Exemplo de arquivo `.env`:

```
MONGODB_URL=mongodb://localhost:27017
MONGODB_DB=auctiondb
```

## Executando o Projeto

### Executando Localmente com Go

1. Certifique-se de que o MongoDB está rodando localmente ou acessível via a variável `MONGODB_URL` que você definiu.
2. Defina as variáveis de ambiente no arquivo `cmd/auction/.env`.
3. A partir da raiz do projeto, execute:

```bash
go run cmd/auction/main.go
```

O servidor será iniciado na porta `8080`.

### Executando com Docker Compose

1. Certifique-se de que o Docker e o Docker Compose estão instalados.
2. Defina as variáveis de ambiente no arquivo `cmd/auction/.env`.
3. A partir da raiz do projeto, execute:

```bash
docker-compose up --build
```

Isso iniciará tanto a aplicação quanto um container MongoDB. A aplicação estará acessível na porta `8080`.

Para parar os containers, pressione `Ctrl+C` e depois execute:

```bash
docker-compose down
```

## Executando os Testes

Para executar os testes, use o seguinte comando a partir da raiz do projeto:

```bash
go test ./...
```

Isso executará todos os testes do projeto.

## Estrutura do Projeto (Resumo)

- `cmd/auction/`: Ponto de entrada principal da aplicação e arquivos de ambiente.
- `configuration/`: Arquivos de configuração para banco de dados e logger.
- `internal/`: Lógica principal da aplicação incluindo entidades, casos de uso, infraestrutura e controladores.
- `Dockerfile`: Instruções para build da imagem Docker.
- `docker-compose.yml`: Configuração do Docker Compose para a aplicação e MongoDB.

## Notas Adicionais

- A aplicação utiliza o Gin como framework web HTTP.
- MongoDB é usado como banco de dados principal.
- As variáveis de ambiente são carregadas usando `godotenv` a partir do arquivo `.env`.

---

Se você tiver alguma dúvida ou precisar de mais assistência, por favor consulte o código fonte ou entre em contato com o mantenedor do projeto.
go run cmd/auction/main.go
