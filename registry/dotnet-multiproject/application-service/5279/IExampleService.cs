using Example.Contracts.Models;
using Example.Contracts.Requests;

namespace Example.Application.Interfaces;

public interface IItemService
{
    Task<Item> GetById(int id, CancellationToken cancellationToken = default);
    Task<Item> Create(CreateItemRequest request, CancellationToken cancellationToken = default);
    Task ArchiveExpired(CancellationToken cancellationToken = default);
}
