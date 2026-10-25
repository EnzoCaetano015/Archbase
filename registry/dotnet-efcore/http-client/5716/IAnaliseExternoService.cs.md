```csharp
using System.Threading;
using System.Threading.Tasks;

namespace Example.Service.Interfaces;

public interface IAnaliseExternoService
{
    Task<string> Analisar(string texto, CancellationToken cancellationToken);
}
```
