using Example.Contracts.Responses;
using Microsoft.AspNetCore.Diagnostics;

namespace Example.Api.Config;

public sealed class ApiExceptionHandler(ILogger<ApiExceptionHandler> logger) : IExceptionHandler
{
    public async ValueTask<bool> TryHandleAsync(
        HttpContext context,
        Exception exception,
        CancellationToken cancellationToken)
    {
        var status = exception is InvalidOperationException
            ? StatusCodes.Status400BadRequest
            : StatusCodes.Status500InternalServerError;
        var message = status == StatusCodes.Status500InternalServerError
            ? "An unexpected error occurred."
            : exception.Message;

        logger.LogError(exception, "Request failed at {Path}", context.Request.Path);
        context.Response.StatusCode = status;
        await context.Response.WriteAsJsonAsync(
            new ResponseEnvelope<object>(false, Error: message),
            cancellationToken);
        return true;
    }
}
