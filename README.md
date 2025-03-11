

- para rodar o projeto, basta abrir o terminal na raiz do projeto e digitar:
  - docker-compose up --build
- caso haja problemas na subida dos containers é possível também forçar a inicialização dos serviços separadamente:
  - no diretório service-a:
    - docker run -p 8080:8080 service-a
  - no diretório service-b:
    - docker run -p 9090:9090 service-b

- para criar uma requisição basta adicionar o curl abaixo no postman:
  - curl -X POST http://localhost:8080/cep \
    -H "Content-Type: application/json" \
    -d '{"cep": "01001000"}'
