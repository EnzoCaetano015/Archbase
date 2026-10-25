```dart
import 'package:flutter/material.dart';

import 'auth_guard.dart';
import 'pages/items/items_page.dart';

class AppRoutes {
  static const autenticacao = '/autenticacao';
  static const items = '/items';

  static Map<String, WidgetBuilder> get routes => {
    autenticacao: (_) =>
        const Scaffold(body: Center(child: Text('Autenticação'))),
    items: (_) => AuthGuard(builder: (_) => const ItemsPage()),
  };

  static Route<dynamic>? onGenerateRoute(RouteSettings settings) {
    final uri = Uri.tryParse(settings.name ?? '');
    if (uri == null ||
        uri.pathSegments.length != 2 ||
        uri.pathSegments[0] != 'item') {
      return null;
    }
    final id = int.tryParse(uri.pathSegments[1]);
    if (id == null || id <= 0) return null;
    return MaterialPageRoute<void>(
      settings: settings,
      builder: (_) => AuthGuard(
        builder: (_) => Scaffold(
          appBar: AppBar(title: const Text('Detalhe')),
          body: Center(child: Text('Item $id')),
        ),
      ),
    );
  }
}
```
