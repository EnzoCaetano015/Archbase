using Example.Contracts.Models;
using Example.Contracts.Requests;
using Example.Infrastructure.Interfaces;
using PetaPoco;
using PetaPoco.SqlKata;
using SqlKata;

namespace Example.Infrastructure;

public sealed class ItemRepository(IDatabase database) : IItemRepository
{
    public Task<Item?> GetById(int id, CancellationToken cancellationToken = default)
    {
        var query = new Query("Item").Where("Id", id).ToSql();
        return database.FirstOrDefaultAsync<Item>(query);
    }

    public async Task<Item> Create(CreateItemRequest request, CancellationToken cancellationToken = default)
    {
        var id = await database.InsertAsync("Item", request);
        return (await GetById(Convert.ToInt32(id), cancellationToken))!;
    }

    public async Task ArchiveExpired(DateTime cutoff, CancellationToken cancellationToken = default)
    {
        var query = new Query("Item")
            .Where("ExpiresAt", "<=", cutoff)
            .AsUpdate(new { IsArchived = true })
            .ToSql();
        await database.ExecuteAsync(query);
    }
}
