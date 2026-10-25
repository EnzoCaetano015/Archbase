```csharp
using System.Threading;
using System.Threading.Tasks;
using Example.Model;

namespace Example.Repository.Interfaces;

public interface IItemRepository
{
    Task<Item?> ObterPorId(int id, CancellationToken cancellationToken);
    Task Inserir(Item item, CancellationToken cancellationToken);
}
```
