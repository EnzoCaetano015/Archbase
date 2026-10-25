```dart
class CreateItemRequest {
  final String titulo;

  const CreateItemRequest({required this.titulo});

  Map<String, dynamic> toJson() => {'titulo': titulo};

  factory CreateItemRequest.fromJson(Map<String, dynamic> json) =>
      CreateItemRequest(titulo: json['titulo'] as String);
}
```
