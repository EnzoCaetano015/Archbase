```csharp
using System.Net;

namespace Example.Model.Response;

public class Response
{
    public bool Sucesso { get; set; }
    public string Mensagem { get; set; } = string.Empty;
    public HttpStatusCode StatusCode { get; set; }
}

public class Response<T> : Response
{
    public T? Dados { get; set; }
}
```
