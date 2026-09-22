from fastapi import Depends

from app.core.database import Database
from app.modules.example.repository import ExampleRepository
from app.modules.example.schemas import ExampleCreate, ExampleResponse
from app.utils.dependencies_helpers import get_database


class ExampleService:
    def __init__(self, repository: ExampleRepository) -> None:
        self._repository = repository

    def create(self, payload: ExampleCreate) -> ExampleResponse:
        example_id, name = self._repository.create(payload.name)
        return ExampleResponse(id=example_id, name=name)


def get_example_service(
    database: Database = Depends(get_database),
) -> ExampleService:
    return ExampleService(ExampleRepository(database))
