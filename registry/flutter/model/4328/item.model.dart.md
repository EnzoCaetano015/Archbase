```dart
class Item {
  final int id;
  final String titulo;
  final DateTime dataCadastro;
  final DateTime? dataAlteracao;

  const Item({
    required this.id,
    required this.titulo,
    required this.dataCadastro,
    this.dataAlteracao,
  });

  factory Item.fromMap(Map<String, dynamic> map) => Item(
    id: map['id'] as int,
    titulo: map['titulo'] as String,
    dataCadastro: DateTime.parse(map['dataCadastro'] as String),
    dataAlteracao: map['dataAlteracao'] == null
        ? null
        : DateTime.parse(map['dataAlteracao'] as String),
  );

  Map<String, dynamic> toJson() => {
    'id': id,
    'titulo': titulo,
    'dataCadastro': dataCadastro.toIso8601String(),
    'dataAlteracao': dataAlteracao?.toIso8601String(),
  };
}
```
