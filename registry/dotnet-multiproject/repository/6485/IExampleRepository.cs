using Example.Contracts.Models;
using Example.Contracts.Requests;

namespace Example.Infrastructure.Interfaces;

public interface IItemRepository
{
    Task<Item?> GetById(int id, CancellationToken cancellationToken = default);
    Task<Item> Create(CreateItemRequest request, CancellationToken cancellationToken = default);
    Task ArchiveExpired(DateTime cutoff, CancellationToken cancellationToken = default);
}
