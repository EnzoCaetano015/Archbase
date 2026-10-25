```csharp
using System;
using System.Threading;
using System.Threading.Tasks;
using Example.Model;
using Example.Repository.Interfaces;
using Microsoft.Data.SqlClient;
using Microsoft.EntityFrameworkCore;

namespace Example.Repository;

public class ItemRepository : IItemRepository
{
    private readonly ApplicationDbContext _context;
    public ItemRepository(ApplicationDbContext context) => _context = context;

    public Task<Item?> ObterPorId(int id, CancellationToken cancellationToken) =>
        _context.Item
            .FromSqlRaw("SELECT * FROM Item WHERE Id = @Id", new SqlParameter("@Id", id))
            .AsNoTracking()
            .FirstOrDefaultAsync(cancellationToken);

    public async Task Inserir(Item item, CancellationToken cancellationToken)
    {
        await using var transaction = await _context.Database.BeginTransactionAsync(cancellationToken);
        try
        {
            const string sql = "INSERT INTO Item (Titulo, DataCadastro, DataAlteracao) " +
                "VALUES (@Titulo, @DataCadastro, @DataAlteracao)";
            await _context.Database.ExecuteSqlRawAsync(sql, new object[]
            {
                new SqlParameter("@Titulo", item.Titulo),
                new SqlParameter("@DataCadastro", item.DataCadastro),
                new SqlParameter("@DataAlteracao", (object?)item.DataAlteracao ?? DBNull.Value)
            }, cancellationToken);
            await transaction.CommitAsync(cancellationToken);
        }
        catch
        {
            await transaction.RollbackAsync(CancellationToken.None);
            throw;
        }
    }
}
```
