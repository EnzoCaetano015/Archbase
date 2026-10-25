```csharp
using System;
using System.Net;
using System.Threading;
using System.Threading.Tasks;
using Example.Model;
using Example.Model.Request;
using Example.Model.Response;
using Example.Repository.Interfaces;
using Example.Service.Interfaces;
using Microsoft.Extensions.Logging;

namespace Example.Service;

public class ItemService : IItemService
{
    private readonly IItemRepository _repository;
    private readonly ILogger<ItemService> _logger;

    public ItemService(IItemRepository repository, ILogger<ItemService> logger)
    {
        _repository = repository;
        _logger = logger;
    }

    public async Task<Response<Item>> ObterPorId(int id, CancellationToken cancellationToken)
    {
        if (id <= 0) return new Response<Item>
        {
            Mensagem = "Identificador inválido.", StatusCode = HttpStatusCode.BadRequest
        };
        try
        {
            var item = await _repository.ObterPorId(id, cancellationToken);
            return item is null
                ? new Response<Item> { Mensagem = "Item não encontrado.", StatusCode = HttpStatusCode.NotFound }
                : new Response<Item> { Sucesso = true, Mensagem = "Item obtido.", Dados = item, StatusCode = HttpStatusCode.OK };
        }
        catch (OperationCanceledException) { throw; }
        catch (Exception error)
        {
            _logger.LogError(error, "Falha ao obter item {Id}", id);
            return new Response<Item> { Mensagem = "Não foi possível obter o item.", StatusCode = HttpStatusCode.InternalServerError };
        }
    }

    public async Task<Response> Criar(Request<CriarItemRequest> request, CancellationToken cancellationToken)
    {
        var titulo = request.Dados?.Titulo;
        if (string.IsNullOrWhiteSpace(titulo) || titulo.Length > 200)
            return new Response { Mensagem = "Título inválido.", StatusCode = HttpStatusCode.BadRequest };
        try
        {
            await _repository.Inserir(new Item { Titulo = titulo, DataCadastro = DateTime.UtcNow }, cancellationToken);
            return new Response { Sucesso = true, Mensagem = "Item criado.", StatusCode = HttpStatusCode.Created };
        }
        catch (OperationCanceledException) { throw; }
        catch (Exception error)
        {
            _logger.LogError(error, "Falha ao criar item");
            return new Response { Mensagem = "Não foi possível criar o item.", StatusCode = HttpStatusCode.InternalServerError };
        }
    }
}
```
