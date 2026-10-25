```dart
class ItemStatusDTO {
  final int id;
  final String titulo;
  final bool concluido;

  const ItemStatusDTO({
    required this.id,
    required this.titulo,
    required this.concluido,
  });

  factory ItemStatusDTO.fromMap(Map<String, dynamic> map) => ItemStatusDTO(
    id: map['id'] as int,
    titulo: map['titulo'] as String,
    concluido: map['concluido'] as bool,
  );

  Map<String, dynamic> toJson() => {
    'id': id,
    'titulo': titulo,
    'concluido': concluido,
  };
}
```
