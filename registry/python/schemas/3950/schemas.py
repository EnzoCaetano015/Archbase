from pydantic import BaseModel, Field


class ExampleCreate(BaseModel):
    name: str = Field(min_length=1, max_length=120)


class ExampleResponse(BaseModel):
    id: int
    name: str
