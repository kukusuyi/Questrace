import '../../shared/widgets/lan_computer_picker.dart';
import '../profile/profile_card.dart';
import '../../core/update/update_preferences.dart';
import '../update/update_check.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import '../../core/config/effective_api_base_url.dart';
import '../../core/storage/app_settings_controller.dart';
import '../../shared/widgets/appearance_card.dart';
import '../../shared/widgets/server_status_card.dart';
import '../auth/auth_controller.dart';

class SettingsPage extends ConsumerStatefulWidget {
  const SettingsPage({super.key});
  @override
  ConsumerState<SettingsPage> createState() => _SettingsPageState();
}

class _SettingsPageState extends ConsumerState<SettingsPage> {
  final _address = TextEditingController();
  bool _hydrated = false, _saving = false;
  String? _error;
  @override
  void dispose() {
    _address.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final current = ref.watch(effectiveApiBaseUrlProvider);
    if (!_hydrated) {
      _address.text = current;
      _hydrated = true;
    }
    return Scaffold(
      appBar: AppBar(title: const Text('我的')),
      body: ListView(
        padding: const EdgeInsets.all(20),
        keyboardDismissBehavior: ScrollViewKeyboardDismissBehavior.onDrag,
        children: [
          const ProfileCard(),
          const SizedBox(height: 16),
          const AppearanceCard(),
          const SizedBox(height: 16),
          Card(
            child: ExpansionTile(
              shape: const Border(),
              title: const Text('电脑连接'),
              subtitle: Text(
                current.isEmpty ? '尚未连接' : current,
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
              ),
              childrenPadding: const EdgeInsets.all(20),
              children: [
                const Text('切换电脑将退出当前账户。每台电脑的草稿会分别保留。'),
                const LanComputerPicker(),
                const SizedBox(height: 16),
                TextField(
                  controller: _address,
                  keyboardType: TextInputType.url,
                  autocorrect: false,
                  decoration: InputDecoration(
                    labelText: '电脑服务地址',
                    hintText: 'http://192.168.1.10:8080',
                    errorText: _error,
                  ),
                ),
                const SizedBox(height: 12),
                FilledButton(
                  onPressed: _saving ? null : _save,
                  child: Text(_saving ? '正在验证连接…' : '验证并连接'),
                ),
              ],
            ),
          ),
          const SizedBox(height: 16),
          const ServerStatusCard(),
          const SizedBox(height: 16),
          Card(
            child: Padding(
              padding: const EdgeInsets.all(20),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    '题迹 Questrace',
                    style: Theme.of(context).textTheme.titleLarge,
                  ),
                  const SizedBox(height: 8),
                  const Text('让每一道错题都有收获'),
                  SwitchListTile(
                    title: const Text('自动检查更新'),
                    value: ref.watch(autoUpdateProvider),
                    onChanged: ref.read(autoUpdateProvider.notifier).setEnabled,
                  ),
                  TextButton(
                    onPressed: () =>
                        checkForUpdates(context, ref, manual: true),
                    child: const Text('检查更新'),
                  ),
                  const SizedBox(height: 8),
                  const Text('题目与图片保存在电脑上。模型服务与用户管理请在电脑网页设置中配置。'),
                  const SizedBox(height: 16),
                  OutlinedButton(
                    onPressed: () async {
                      await ref.read(authControllerProvider.notifier).logout();
                      if (context.mounted) context.go('/auth');
                    },
                    child: const Text('退出当前账户'),
                  ),
                ],
              ),
            ),
          ),
        ],
      ),
    );
  }

  Future<void> _save() async {
    setState(() {
      _saving = true;
      _error = null;
    });
    try {
      await ref
          .read(appSettingsControllerProvider.notifier)
          .setApiBaseUrlOverride(_address.text);
      if (mounted) context.go('/auth');
    } catch (_) {
      if (mounted) setState(() => _error = '连接失败，请检查完整地址、电脑程序与局域网。');
    } finally {
      if (mounted) setState(() => _saving = false);
    }
  }
}
