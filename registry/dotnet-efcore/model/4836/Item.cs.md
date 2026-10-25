```csharp
using System;

namespace Example.Model;

public class Item
{
    public int Id { get; set; }
    public string Titulo { get; set; } = string.Empty;
    public DateTime DataCadastro { get; set; }
    public DateTime? DataAlteracao { get; set; }
}
```
