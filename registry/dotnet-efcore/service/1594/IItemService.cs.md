```csharp
using System.Threading;
using System.Threading.Tasks;
using Example.Model;
using Example.Model.Request;
using Example.Model.Response;

namespace Example.Service.Interfaces;

public interface IItemService
{
    Task<Response<Item>> ObterPorId(int id, CancellationToken cancellationToken);
    Task<Response> Criar(Request<CriarItemRequest> request, CancellationToken cancellationToken);
}
```
