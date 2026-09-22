from app.core.database import Database


class ExampleRepository:
    def __init__(self, database: Database) -> None:
        self._database = database

    def create(self, name: str) -> tuple[int, str]:
        with self._database.transaction() as session:
            with session.cursor() as cursor:
                cursor.execute(
                    """
                    INSERT INTO example (name)
                    VALUES (%s)
                    RETURNING id, name
                    """,
                    (name,),
                )
                row = cursor.fetchone()

        if row is None:
            raise RuntimeError("insert did not return the created example")

        return int(row[0]), str(row[1])
