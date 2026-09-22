using Example.Application.Interfaces;
using Example.Contracts.Requests;
using Microsoft.AspNetCore.Authorization;
using Microsoft.AspNetCore.Mvc;

namespace Example.Api.Controllers;

[ApiController]
[Route("items")]
[Authorize]
public sealed class ItemsController(IItemService itemService) : ControllerBase
{
    [HttpGet("{id:int}")]
    public async Task<IActionResult> GetById(int id, CancellationToken cancellationToken)
    {
        var item = await itemService.GetById(id, cancellationToken);
        return Ok(item);
    }

    [HttpPost]
    public async Task<IActionResult> Create(CreateItemRequest request, CancellationToken cancellationToken)
    {
        var item = await itemService.Create(request, cancellationToken);
        return CreatedAtAction(nameof(GetById), new { id = item.Id }, item);
    }
}
