```csharp
namespace Example.Model.Request;

public class Request<T> where T : class
{
    public T? Dados { get; set; }
}
```
