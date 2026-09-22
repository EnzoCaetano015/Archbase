namespace Example.Jobs.Interfaces;

public interface IJob
{
    Task Execute(CancellationToken cancellationToken = default);
}
