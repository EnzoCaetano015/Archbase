```csharp
using Example.Model;
using Example.Model.DTO;
using Microsoft.EntityFrameworkCore;

namespace Example.Repository;

public class ApplicationDbContext : DbContext
{
    public ApplicationDbContext(DbContextOptions<ApplicationDbContext> options)
        : base(options) { }

    public DbSet<Item> Item => Set<Item>();
    public DbSet<ItemResumoDTO> ItemResumoDTO => Set<ItemResumoDTO>();

    protected override void OnModelCreating(ModelBuilder modelBuilder)
    {
        base.OnModelCreating(modelBuilder);
        modelBuilder.Entity<Item>().ToTable("Item");
        modelBuilder.Entity<ItemResumoDTO>().HasNoKey();
    }
}
```
