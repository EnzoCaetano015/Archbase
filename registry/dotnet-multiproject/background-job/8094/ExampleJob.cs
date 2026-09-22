using Example.Application.Interfaces;
using Example.Jobs.Attributes;
using Example.Jobs.Interfaces;
using Microsoft.Extensions.Logging;

namespace Example.Jobs;

[Job("0 0 * * *", "archive-items")]
public sealed class ArchiveItemsJob(IItemService itemService, ILogger<ArchiveItemsJob> logger) : IJob
{
    public async Task Execute(CancellationToken cancellationToken = default)
    {
        logger.LogInformation("Starting scheduled item archival");
        await itemService.ArchiveExpired(cancellationToken);
        logger.LogInformation("Finished scheduled item archival");
    }
}
