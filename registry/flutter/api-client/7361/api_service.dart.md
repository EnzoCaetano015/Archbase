```dart
import 'dart:convert';

import 'package:http/http.dart' as http;

import '../notificacao_provider.dart';

class ApiService {
  final http.Client client;
  final Uri baseUrl;
  final AuthProvider auth;
  Future<void> _fila = Future<void>.value();

  ApiService({required this.client, required this.baseUrl, required this.auth});

  Future<http.Response> requestAutenticada(
    String path, {
    String metodo = 'GET',
    Map<String, dynamic>? corpo,
  }) {
    final resultado = _fila.then((_) async {
      final token = auth.token;
      if (token == null || token.isEmpty) throw StateError('Sessão ausente.');
      final headers = {
        'Content-Type': 'application/json',
        'Authorization': 'Bearer $token',
      };
      final uri = baseUrl.resolve(path);
      if (uri.origin != baseUrl.origin) throw ArgumentError('Origem inválida.');
      final http.Response response;
      switch (metodo.toUpperCase()) {
        case 'GET':
          response = await client.get(uri, headers: headers);
          break;
        case 'POST':
          response = await client.post(
            uri,
            headers: headers,
            body: jsonEncode(corpo),
          );
          break;
        default:
          throw ArgumentError('Método não suportado.');
      }
      if (response.statusCode == 401) await auth.logout();
      return response;
    });
    // A failed request must not prevent queued futures from completing.
    _fila = resultado.then<void>((_) {}, onError: (Object _, StackTrace __) {});
    return resultado;
  }
}
```
