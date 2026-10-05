import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../core/network/api_client.dart';
import '../../core/network/api_exception.dart';
import '../../shared/widgets/subject_picker.dart';
import '../auth/auth_controller.dart';

final userProfileProvider = FutureProvider<Map<String, dynamic>>((ref) async {
  ref.watch(authControllerProvider.select((state) => state.session));
  final response = await ref
      .watch(apiClientProvider)
      .get<Map<String, dynamic>>('/api/v1/users/me');
  return response.data ?? {};
});

class ProfileCard extends ConsumerStatefulWidget {
  const ProfileCard({super.key});
  @override
  ConsumerState<ProfileCard> createState() => _ProfileCardState();
}

class _ProfileCardState extends ConsumerState<ProfileCard> {
  String? stage;
  bool busy = false;
  String? error;
  @override
  Widget build(BuildContext context) {
    final profile = ref.watch(userProfileProvider);
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(20),
        child: profile.when(
          loading: () => const LinearProgressIndicator(),
          error: (e, s) => TextButton(
            onPressed: () => ref.invalidate(userProfileProvider),
            child: const Text('个人信息加载失败，点击重试'),
          ),
          data: (data) {
            final saved = data['education_stage'] as String? ?? 'university';
            return Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                Text('个人信息', style: Theme.of(context).textTheme.titleLarge),
                const SizedBox(height: 8),
                Text('${data['username'] ?? ''} · ${data['email'] ?? ''}'),
                const SizedBox(height: 16),
                DropdownButtonFormField<String>(
                  key: ValueKey(stage ?? saved),
                  initialValue: stage ?? saved,
                  decoration: const InputDecoration(labelText: '当前学习阶段'),
                  items: const [
                    DropdownMenuItem(value: 'university', child: Text('大学')),
                    DropdownMenuItem(value: 'highschool', child: Text('高中')),
                  ],
                  onChanged: busy
                      ? null
                      : (value) => setState(() => stage = value),
                ),
                const SizedBox(height: 12),
                const Text('按阶段推荐学科，任何专业科目都可添加。修改阶段不会改变已有题目的分类。'),
                if (error != null)
                  Text(
                    error!,
                    style: TextStyle(
                      color: Theme.of(context).colorScheme.error,
                    ),
                  ),
                const SizedBox(height: 12),
                FilledButton(
                  onPressed: busy || stage == null || stage == saved
                      ? null
                      : () async {
                          setState(() {
                            busy = true;
                            error = null;
                          });
                          try {
                            await ref
                                .read(apiClientProvider)
                                .put(
                                  '/api/v1/users/me',
                                  data: {'education_stage': stage},
                                );
                            if (!mounted) return;
                            ref.invalidate(userProfileProvider);
                            ref.invalidate(subjectCatalogProvider);
                            if (mounted) {
                              setState(() => stage = null);
                              ScaffoldMessenger.of(this.context).showSnackBar(
                                const SnackBar(content: Text('学习阶段已保存')),
                              );
                            }
                          } catch (e) {
                            if (mounted) {
                              setState(() => error = describeError(e));
                            }
                          } finally {
                            if (mounted) setState(() => busy = false);
                          }
                        },
                  child: Text(busy ? '正在保存…' : '保存学习阶段'),
                ),
              ],
            );
          },
        ),
      ),
    );
  }
}
