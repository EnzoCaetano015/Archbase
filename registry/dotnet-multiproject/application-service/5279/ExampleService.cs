using Example.Application.Interfaces;
using Example.Contracts.Models;
using Example.Contracts.Requests;
using Example.Infrastructure.Interfaces;

namespace Example.Application;

public sealed class ItemService(IItemRepository repository) : IItemService
{
    public async Task<Item> GetById(int id, CancellationToken cancellationToken = default)
    {
        return await repository.GetById(id, cancellationToken)
            ?? throw new InvalidOperationException("Item not found");
    }

    public Task<Item> Create(CreateItemRequest request, CancellationToken cancellationToken = default)
    {
        return repository.Create(request, cancellationToken);
    }

    public Task ArchiveExpired(CancellationToken cancellationToken = default)
    {
        return repository.ArchiveExpired(DateTime.UtcNow, cancellationToken);
    }
}
