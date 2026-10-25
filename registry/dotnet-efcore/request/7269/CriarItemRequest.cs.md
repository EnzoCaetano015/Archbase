```csharp
using System.ComponentModel.DataAnnotations;

namespace Example.Model.Request;

public class CriarItemRequest
{
    [Required, StringLength(200)]
    public string Titulo { get; set; } = string.Empty;
}
```
