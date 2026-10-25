```dart
import 'package:flutter/material.dart';

class ItemCard extends StatelessWidget {
  final String titulo;
  final VoidCallback onTap;

  const ItemCard({super.key, required this.titulo, required this.onTap});

  @override
  Widget build(BuildContext context) => Card(
    child: ListTile(
      title: Text(titulo, maxLines: 1, overflow: TextOverflow.ellipsis),
      onTap: onTap,
    ),
  );
}
```
