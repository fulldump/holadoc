# InceptionDB

This is inceptiondb in markdown

## ¿Qué es InceptionDB?

InceptionDB es un sistema de base de datos NoSQL orientado a documentos.

En lugar de guardar los datos en tablas (como en las bases de datos relacionales), guarda estructuras de datos JSON en 
colecciones con un esquema dinámico, haciendo que la integración de los datos en las aplicaciones sea más fácil.

Está completamente implementado en Go (el lenguaje de programación de Google) por lo que está disponible para prácticamente
cualquier sistema operativo y arquitectura, podrías instalar InceptionDB en tu móvil.

## Características principales

### Consultas

### Indexación

### Replicación

### Ejecución de JavaScript

## Ejemplo de código

```go
func Hello() string {
	return "Hello"
}
```

Y esto es otro bloque de código:

<code lang="javascript">
function Hello() {
    return "Hello"
}
</code>


```http request
GET / HTTP/1.1
Host: www.example.com
User-Agent: Mozilla/5.0
Accept: text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8
Accept-Language: en-GB,en;q=0.5
Accept-Encoding: gzip, deflate, br
Connection: keep-alive

Hello WOrld
```


```http response
HTTP/1.1 200 OK
Date: Mon, 23 May 2005 22:38:34 GMT
Content-Type: text/html; charset=UTF-8
Content-Length: 155
Last-Modified: Wed, 08 Jan 2003 23:11:55 GMT
Server: Apache/1.3.3.7 (Unix) (Red-Hat/Linux)
ETag: "3f80f-1b6-3e1cb03b"
Accept-Ranges: bytes
Connection: close

<html>
  <head>
    <title>An Example Page</title>
  </head>
  <body>
    <p>Hello World, this is a very simple HTML document.</p>
  </body>
</html>
```

```http response
HTTP/1.1 200 OK
Date: Mon, 23 May 2005 22:38:34 GMT
Content-Type: application/json; charset=UTF-8
Content-Length: 155
Last-Modified: Wed, 08 Jan 2003 23:11:55 GMT
Server: Apache/1.3.3.7 (Unix) (Red-Hat/Linux)
ETag: "3f80f-1b6-3e1cb03b"
Accept-Ranges: bytes
Connection: close

{
    "hello": "world",
    "numbers": [1,2,3]
}
```