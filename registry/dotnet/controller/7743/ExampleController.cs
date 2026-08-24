using Microsoft.AspNetCore.Mvc;

namespace Example.Api.Controllers;

[ApiController]
[Route("api/[controller]")]
public sealed class ExampleController(IExampleService service) : ControllerBase
{
    [HttpGet]
    public async Task<ActionResult<IReadOnlyList<ExampleResponse>>> GetAsync(
        CancellationToken cancellationToken)
    {
        var result = await service.ListAsync(cancellationToken);

        return Ok(result);
    }
}
