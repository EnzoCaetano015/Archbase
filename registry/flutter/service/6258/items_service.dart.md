```dart
import 'dart:convert';

import '../schemas/item.model.dart';
import 'api_service.dart';

class ItemsService {
  final ApiService apiService;
  ItemsService(this.apiService);

  Future<List<Item>> listar() async {
    final response = await apiService.requestAutenticada('/app/items');
    if (response.statusCode != 200) throw StateError('Falha ao obter itens.');
    final data = jsonDecode(response.body) as Map<String, dynamic>;
    if (data['sucesso'] != true) throw StateError('Operação não concluída.');
    return (data['dados'] as List<dynamic>)
        .map((value) => Item.fromMap(value as Map<String, dynamic>))
        .toList();
  }
}
```
