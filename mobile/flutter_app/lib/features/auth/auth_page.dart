import '../../shared/widgets/lan_computer_picker.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../core/config/effective_api_base_url.dart';
import '../../core/storage/app_settings_controller.dart';
import '../../core/config/app_environment.dart';
import '../../core/network/server_capabilities.dart';
import '../../shared/models/auth_models.dart';
import 'auth_controller.dart';

class AuthPage extends ConsumerStatefulWidget {
  const AuthPage({super.key});

  @override
  ConsumerState<AuthPage> createState() => _AuthPageState();
}

class _AuthPageState extends ConsumerState<AuthPage> {
  final _usernameController = TextEditingController();
  final _passwordController = TextEditingController();
  final _emailController = TextEditingController();
  final _apiUrlController = TextEditingController();
  bool _registerMode = false;
  String? _educationStage;
  bool _apiUrlExpanded = false;
  bool _connecting = false;
  String? _connectionError;
  bool _apiUrlHydrated = false;
  String? _lastShownErrorMessage;

  @override
  void initState() {
    super.initState();
    ref.listenManual<AuthState>(authControllerProvider, (previous, next) {
      final message = next.errorMessage?.trim() ?? '';
      if (!mounted || message.isEmpty || message == _lastShownErrorMessage) {
        return;
      }

      _lastShownErrorMessage = message;
      _showMessage(message);
    });
  }

  @override
  void dispose() {
    _usernameController.dispose();
    _passwordController.dispose();
    _emailController.dispose();
    _apiUrlController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final authState = ref.watch(authControllerProvider);
    final environment = ref.watch(appEnvironmentProvider);
    final apiBaseUrl = ref
        .watch(appSettingsControllerProvider)
        .apiBaseUrlOverride;
    final effective = ref.watch(effectiveApiBaseUrlProvider);
    final connecting = effective.isEmpty || _apiUrlExpanded;
    _hydrateApiUrlIfNeeded(environment.defaultApiBaseUrl, apiBaseUrl);
    final registration =
        effective.isNotEmpty &&
        ref
                .watch(serverCapabilitiesProvider)
                .valueOrNull?['registration_enabled'] ==
            true;
    return Scaffold(
      body: SafeArea(
        child: Center(
          child: SingleChildScrollView(
            keyboardDismissBehavior: ScrollViewKeyboardDismissBehavior.onDrag,
            padding: const EdgeInsets.all(24),
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 460),
              child: Card(
                child: Padding(
                  padding: const EdgeInsets.all(24),
                  child: AutofillGroup(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(
                          '题迹 Questrace',
                          style: Theme.of(context).textTheme.headlineMedium
                              ?.copyWith(fontWeight: FontWeight.w700),
                        ),
                        const SizedBox(height: 8),
                        Text(connecting ? '连接你的电脑，开始整理错题。' : '欢迎回来，继续你的学习。'),
                        const SizedBox(height: 28),
                        if (connecting) ...[
                          const LanComputerPicker(),
                          TextField(
                            controller: _apiUrlController,
                            keyboardType: TextInputType.url,
                            autocorrect: false,
                            decoration: InputDecoration(
                              labelText: '电脑服务地址',
                              hintText: 'http://192.168.1.10:8080',
                              errorText: _connectionError,
                            ),
                          ),
                          const SizedBox(height: 12),
                          const Text('打开电脑上的题迹，在“电脑连接”中找到局域网地址。手机与电脑需连接同一网络。'),
                          const SizedBox(height: 20),
                          SizedBox(
                            width: double.infinity,
                            child: FilledButton(
                              onPressed: _connecting
                                  ? null
                                  : () => _saveApiUrl(
                                      environment.defaultApiBaseUrl,
                                    ),
                              child: Text(_connecting ? '正在验证连接…' : '连接电脑'),
                            ),
                          ),
                          if (effective.isNotEmpty)
                            TextButton(
                              onPressed: () =>
                                  setState(() => _apiUrlExpanded = false),
                              child: const Text('返回登录'),
                            ),
                        ] else ...[
                          ListTile(
                            contentPadding: EdgeInsets.zero,
                            leading: const Icon(Icons.computer_outlined),
                            title: const Text('当前电脑'),
                            subtitle: Text(effective),
                            trailing: TextButton(
                              onPressed: () =>
                                  setState(() => _apiUrlExpanded = true),
                              child: const Text('更换'),
                            ),
                          ),
                          const SizedBox(height: 16),
                          TextFormField(
                            controller: _usernameController,
                            autofillHints: const [AutofillHints.username],
                            decoration: const InputDecoration(labelText: '用户名'),
                          ),
                          const SizedBox(height: 16),
                          if (_registerMode) ...[
                            DropdownButtonFormField<String>(
                              initialValue: _educationStage,
                              decoration: const InputDecoration(
                                labelText: '当前学习阶段',
                              ),
                              hint: const Text('请选择大学或高中'),
                              items: const [
                                DropdownMenuItem(
                                  value: 'university',
                                  child: Text('大学'),
                                ),
                                DropdownMenuItem(
                                  value: 'highschool',
                                  child: Text('高中'),
                                ),
                              ],
                              onChanged: (v) =>
                                  setState(() => _educationStage = v),
                            ),
                            const SizedBox(height: 16),
                            TextFormField(
                              controller: _emailController,
                              keyboardType: TextInputType.emailAddress,
                              autofillHints: const [AutofillHints.email],
                              decoration: const InputDecoration(
                                labelText: '邮箱',
                              ),
                            ),
                            const SizedBox(height: 16),
                          ],
                          TextFormField(
                            controller: _passwordController,
                            obscureText: true,
                            autofillHints: const [AutofillHints.password],
                            decoration: const InputDecoration(labelText: '密码'),
                            onFieldSubmitted: (_) => _submit(),
                          ),
                          const SizedBox(height: 20),
                          if ((authState.errorMessage ?? '').isNotEmpty)
                            Padding(
                              padding: const EdgeInsets.only(bottom: 16),
                              child: Text(
                                authState.errorMessage!,
                                style: TextStyle(
                                  color: Theme.of(context).colorScheme.error,
                                ),
                              ),
                            ),
                          SizedBox(
                            width: double.infinity,
                            child: FilledButton(
                              onPressed: authState.isSubmitting
                                  ? null
                                  : _submit,
                              child: Text(
                                authState.isSubmitting
                                    ? '正在登录…'
                                    : _registerMode
                                    ? '创建账户'
                                    : '登录',
                              ),
                            ),
                          ),
                          if (registration)
                            TextButton(
                              onPressed: () => setState(
                                () => _registerMode = !_registerMode,
                              ),
                              child: Text(_registerMode ? '已有账户，去登录' : '创建新账户'),
                            ),
                        ],
                      ],
                    ),
                  ),
                ),
              ),
            ),
          ),
        ),
      ),
    );
  }

  void _hydrateApiUrlIfNeeded(String defaultBaseUrl, String overrideBaseUrl) {
    if (_apiUrlHydrated) return;
    _apiUrlController.text = overrideBaseUrl.isNotEmpty
        ? overrideBaseUrl
        : defaultBaseUrl;
    _apiUrlHydrated = true;
  }

  Future<void> _saveApiUrl(String defaultBaseUrl) async {
    if (_connecting) return;
    setState(() {
      _connecting = true;
      _connectionError = null;
    });
    try {
      await ref
          .read(appSettingsControllerProvider.notifier)
          .setApiBaseUrlOverride(_apiUrlController.text.trim());
      if (mounted) {
        setState(() {
          _apiUrlExpanded = false;
          _apiUrlHydrated = false;
        });
      }
    } catch (_) {
      if (mounted) setState(() => _connectionError = '连接失败，请检查电脑地址、程序和网络。');
    } finally {
      if (mounted) setState(() => _connecting = false);
    }
  }

  Future<void> _submit() async {
    final username = _usernameController.text.trim();
    final password = _passwordController.text.trim();
    final email = _emailController.text.trim();

    if (username.isEmpty || password.isEmpty) {
      _showMessage('请输入用户名和密码');
      return;
    }

    if (_registerMode && _educationStage == null) {
      _showMessage('请选择当前学习阶段：大学或高中');
      return;
    }
    if (_registerMode && email.isEmpty) {
      _showMessage('注册模式下需要填写邮箱');
      return;
    }

    try {
      if (_registerMode) {
        await ref
            .read(authControllerProvider.notifier)
            .register(
              RegisterPayload(
                educationStage: _educationStage!,
                username: username,
                password: password,
                email: email,
              ),
            );
      } else {
        await ref
            .read(authControllerProvider.notifier)
            .login(LoginPayload(username: username, password: password));
      }

      if (mounted) {
        _lastShownErrorMessage = null;
        context.go('/dashboard');
      }
    } catch (_) {}
  }

  void _showMessage(String message) {
    final messenger = ScaffoldMessenger.of(context);
    messenger
      ..hideCurrentSnackBar()
      ..showSnackBar(
        SnackBar(content: Text(message), behavior: SnackBarBehavior.floating),
      );
  }
}
