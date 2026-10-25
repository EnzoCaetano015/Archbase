```csharp
using System.Threading;
using System.Threading.Tasks;
using Example.Model;
using Example.Model.Request;
using Example.Model.Response;
using Example.Service.Interfaces;
using Microsoft.AspNetCore.Authorization;
using Microsoft.AspNetCore.Mvc;

namespace Example.Api.Controllers.APP;

[ApiController]
[Authorize]
[Route("app/[controller]")]
public class ItemController : ControllerBase
{
    private readonly IItemService _service;
    public ItemController(IItemService service) => _service = service;

    [HttpGet("obterPorId/{id:int}")]
    public async Task<ActionResult<Response<Item>>> ObterPorId(int id, CancellationToken cancellationToken)
    {
        var response = await _service.ObterPorId(id, cancellationToken);
        return StatusCode((int)response.StatusCode, response);
    }

    [HttpPost("criar")]
    public async Task<ActionResult<Response>> Criar([FromBody] Request<CriarItemRequest> request, CancellationToken cancellationToken)
    {
        var response = await _service.Criar(request, cancellationToken);
        return StatusCode((int)response.StatusCode, response);
    }
}
```
