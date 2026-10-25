```dart
import 'package:flutter/material.dart';
import 'package:http/http.dart' as http;
import 'package:provider/provider.dart';
import 'package:shared_preferences/shared_preferences.dart';

import 'notificacao_provider.dart';
import 'routes.dart';
import 'services/api_service.dart';
import 'services/items_service.dart';
import 'style/themes.dart';

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();
  final prefs = await SharedPreferences.getInstance();
  const configuredUrl = String.fromEnvironment('API_BASE_URL');
  final baseUrl = Uri.tryParse(configuredUrl);
  if (baseUrl == null || baseUrl.scheme != 'https' || baseUrl.host.isEmpty) {
    throw StateError('API_BASE_URL deve ser uma URL HTTPS.');
  }
  PreferenciaTema.setTema();
  runApp(
    MultiProvider(
      providers: [
        ChangeNotifierProvider(create: (_) => AuthProvider(prefs)),
        Provider<http.Client>(
          create: (_) => http.Client(),
          dispose: (_, client) => client.close(),
        ),
        Provider(
          create: (context) => ApiService(
            client: context.read<http.Client>(),
            baseUrl: baseUrl,
            auth: context.read<AuthProvider>(),
          ),
        ),
        Provider(create: (context) => ItemsService(context.read<ApiService>())),
      ],
      child: const ExampleApp(),
    ),
  );
}

class ExampleApp extends StatefulWidget {
  const ExampleApp({super.key});
  @override
  State<ExampleApp> createState() => _ExampleAppState();
}

class _ExampleAppState extends State<ExampleApp> with WidgetsBindingObserver {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
  }

  @override
  void didChangePlatformBrightness() => PreferenciaTema.setTema();

  @override
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => ValueListenableBuilder<Brightness>(
    valueListenable: PreferenciaTema.tema,
    builder: (_, brightness, __) => MaterialApp(
      theme: PreferenciaTema.build(brightness),
      initialRoute: AppRoutes.items,
      routes: AppRoutes.routes,
      onGenerateRoute: AppRoutes.onGenerateRoute,
    ),
  );
}
```
