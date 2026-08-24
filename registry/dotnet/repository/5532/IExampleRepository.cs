namespace Example.Domain.Repositories;

public interface IExampleRepository
{
    Task<IReadOnlyList<ExampleEntity>> ListAsync(CancellationToken cancellationToken);

    Task<ExampleEntity?> FindAsync(Guid id, CancellationToken cancellationToken);
}
