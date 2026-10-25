```csharp
using System.IO;
using System.Net.Http;
using System.Net.Http.Json;
using System.Text.Json;
using System.Threading;
using System.Threading.Tasks;
using Example.Service.Interfaces;

namespace Example.Service;

public class AnaliseExternoService : IAnaliseExternoService
{
    private readonly HttpClient _httpClient;
    public AnaliseExternoService(HttpClient httpClient) => _httpClient = httpClient;

    public async Task<string> Analisar(string texto, CancellationToken cancellationToken)
    {
        using var response = await _httpClient.PostAsJsonAsync("analisar", new { texto }, cancellationToken);
        response.EnsureSuccessStatusCode();
        var envelope = await response.Content.ReadFromJsonAsync<JsonElement>(cancellationToken: cancellationToken);
        if (!envelope.TryGetProperty("status", out var status) || status.GetString() != "ok" ||
            !envelope.TryGetProperty("dados", out var dados) ||
            !dados.TryGetProperty("feedback", out var feedback) || feedback.ValueKind != JsonValueKind.String)
            throw new InvalidDataException("Resposta externa inválida.");
        return feedback.GetString()!;
    }
}
```
