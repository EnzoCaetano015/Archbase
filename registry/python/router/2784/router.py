from fastapi import APIRouter, Depends, status

from app.modules.example.schemas import ExampleCreate, ExampleResponse
from app.modules.example.service import ExampleService, get_example_service


router = APIRouter(prefix="/examples", tags=["Examples"])


@router.post(
    "",
    response_model=ExampleResponse,
    status_code=status.HTTP_201_CREATED,
)
def create_example(
    payload: ExampleCreate,
    service: ExampleService = Depends(get_example_service),
) -> ExampleResponse:
    return service.create(payload)
