import 'package:flutter/material.dart';
import 'main.dart' show AppColors;
import 'services/api_service.dart';

/// Screen for Master Admin Control Panel.
class MasterAdminScreen extends StatefulWidget {
  const MasterAdminScreen({super.key});

  @override
  State<MasterAdminScreen> createState() => _MasterAdminScreenState();
}

class _MasterAdminScreenState extends State<MasterAdminScreen>
    with SingleTickerProviderStateMixin {
  late TabController _tabController;
  final ApiService _apiService = ApiService();

  final TextEditingController _emailController = TextEditingController();
  final TextEditingController _notesController = TextEditingController();

  List<Map<String, dynamic>> _whitelistedEmails = [];
  List<Map<String, dynamic>> _pendingKeywords = [];
  List<Map<String, dynamic>> _registeredUsers = [];
  bool _isLoading = true;

  @override
  void initState() {
    super.initState();
    _tabController = TabController(length: 3, vsync: this);
    _loadAdminData();
  }

  @override
  void dispose() {
    _tabController.dispose();
    _emailController.dispose();
    _notesController.dispose();
    super.dispose();
  }

  Future<void> _loadAdminData() async {
    setState(() {
      _isLoading = true;
    });

    final emails = await _apiService.fetchWhitelistedEmails();
    final keywords = await _apiService.fetchPendingKeywords();
    final users = await _apiService.fetchUsersForAdmin();

    if (!mounted) return;

    setState(() {
      _whitelistedEmails = emails;
      _pendingKeywords = keywords;
      _registeredUsers = users;
      _isLoading = false;
    });
  }

  Future<void> _toggleUserAIMatching(String userId, bool enabled) async {
    final success = await _apiService.toggleUserAIMatching(userId, enabled);

    if (!mounted) return;

    if (success) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(
            enabled
                ? 'AI Job Matching enabled for user'
                : 'AI Job Matching disabled for user',
          ),
        ),
      );
      _loadAdminData();
    } else {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Failed to update user AI matching state')),
      );
    }
  }

  Future<void> _addWhitelistedEmail() async {
    final email = _emailController.text.trim();
    if (email.isEmpty) return;

    final success = await _apiService.addWhitelistedEmail(
      email,
      _notesController.text.trim(),
    );

    if (!mounted) return;

    if (success) {
      _emailController.clear();
      _notesController.clear();
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('Added $email to whitelist')),
      );
      _loadAdminData();
    } else {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Failed to add email to whitelist')),
      );
    }
  }

  Future<void> _removeWhitelistedEmail(String id, String email) async {
    final success = await _apiService.deleteWhitelistedEmail(id);

    if (!mounted) return;

    if (success) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('Removed $email from whitelist')),
      );
      _loadAdminData();
    }
  }

  Future<void> _processKeyword(String id, bool approve) async {
    final success = await _apiService.approveKeyword(id, approve);

    if (!mounted) return;

    if (success) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(
            approve ? 'Approved keyword recommendation' : 'Rejected keyword',
          ),
        ),
      );
      _loadAdminData();
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: _buildAppBar(),
      body: _isLoading
          ? const Center(child: CircularProgressIndicator())
          : TabBarView(
              controller: _tabController,
              children: [
                _buildUsersTab(),
                _buildWhitelistTab(),
                _buildKeywordQueueTab(),
              ],
            ),
    );
  }

  PreferredSizeWidget _buildAppBar() {
    return AppBar(
      backgroundColor: AppColors.surface,
      elevation: 0,
      scrolledUnderElevation: 0,
      title: const Text(
        'Master Admin Control',
        style: TextStyle(
          color: AppColors.primary,
          fontSize: 20,
          fontWeight: FontWeight.bold,
        ),
      ),
      bottom: TabBar(
        controller: _tabController,
        labelColor: AppColors.primary,
        indicatorColor: AppColors.primary,
        isScrollable: true,
        tabAlignment: TabAlignment.start,
        tabs: const [
          Tab(text: 'User AI Search'),
          Tab(text: 'Access Whitelist'),
          Tab(text: 'Keywords Queue'),
        ],
      ),
    );
  }

  Widget _buildWhitelistTab() {
    return SingleChildScrollView(
      padding: const EdgeInsets.all(20),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Container(
            padding: const EdgeInsets.all(16),
            decoration: BoxDecoration(
              color: AppColors.surfaceContainerLowest,
              borderRadius: BorderRadius.circular(12),
              border: Border.all(color: AppColors.outlineVariant),
            ),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                const Text(
                  'Add New Whitelisted User',
                  style: TextStyle(
                    fontSize: 16,
                    fontWeight: FontWeight.bold,
                    color: AppColors.primary,
                  ),
                ),
                const SizedBox(height: 12),
                TextField(
                  controller: _emailController,
                  keyboardType: TextInputType.emailAddress,
                  decoration: const InputDecoration(
                    labelText: 'User Email Address',
                    border: OutlineInputBorder(),
                    isDense: true,
                  ),
                ),
                const SizedBox(height: 12),
                TextField(
                  controller: _notesController,
                  decoration: const InputDecoration(
                    labelText: 'Notes / User Role (Optional)',
                    border: OutlineInputBorder(),
                    isDense: true,
                  ),
                ),
                const SizedBox(height: 16),
                SizedBox(
                  width: double.infinity,
                  child: ElevatedButton.icon(
                    onPressed: _addWhitelistedEmail,
                    icon: const Icon(Icons.person_add),
                    label: const Text('Add to Access Whitelist'),
                    style: ElevatedButton.styleFrom(
                      backgroundColor: AppColors.primary,
                      foregroundColor: Colors.white,
                      padding: const EdgeInsets.symmetric(vertical: 12),
                    ),
                  ),
                ),
              ],
            ),
          ),
          const SizedBox(height: 24),
          const Text(
            'Whitelisted Accounts',
            style: TextStyle(
              fontSize: 18,
              fontWeight: FontWeight.bold,
              color: AppColors.primary,
            ),
          ),
          const SizedBox(height: 12),
          ListView.builder(
            shrinkWrap: true,
            physics: const NeverScrollableScrollPhysics(),
            itemCount: _whitelistedEmails.length,
            itemBuilder: (context, index) {
              final item = _whitelistedEmails[index];
              return Card(
                margin: const EdgeInsets.only(bottom: 8),
                shape: RoundedRectangleBorder(
                  borderRadius: BorderRadius.circular(8),
                  side: const BorderSide(color: AppColors.outlineVariant),
                ),
                child: ListTile(
                  leading: const Icon(Icons.verified_user, color: AppColors.successGreen),
                  title: Text(item['email'] ?? ''),
                  subtitle: Text(item['notes']?.isEmpty ?? true ? 'No notes' : item['notes']),
                  trailing: IconButton(
                    icon: const Icon(Icons.delete_outline, color: AppColors.error),
                    onPressed: () => _removeWhitelistedEmail(
                      item['id'] as String,
                      item['email'] as String,
                    ),
                  ),
                ),
              );
            },
          ),
        ],
      ),
    );
  }

  Widget _buildKeywordQueueTab() {
    return SingleChildScrollView(
      padding: const EdgeInsets.all(20),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const Text(
            'Pending Keyword Recommendations from Gemini Flash',
            style: TextStyle(
              fontSize: 16,
              fontWeight: FontWeight.bold,
              color: AppColors.primary,
            ),
          ),
          const SizedBox(height: 8),
          const Text(
            'Extracted from new candidate CVs and bio text boxes. Approved keywords update the master dictionary.',
            style: TextStyle(
              fontSize: 13,
              color: AppColors.onSurfaceVariant,
            ),
          ),
          const SizedBox(height: 16),
          _pendingKeywords.isEmpty
              ? const Padding(
                  padding: EdgeInsets.all(32),
                  child: Center(
                    child: Text('No pending keyword suggestions for review.'),
                  ),
                )
              : ListView.builder(
                  shrinkWrap: true,
                  physics: const NeverScrollableScrollPhysics(),
                  itemCount: _pendingKeywords.length,
                  itemBuilder: (context, index) {
                    final item = _pendingKeywords[index];
                    return Card(
                      margin: const EdgeInsets.only(bottom: 8),
                      shape: RoundedRectangleBorder(
                        borderRadius: BorderRadius.circular(8),
                        side: const BorderSide(color: AppColors.outlineVariant),
                      ),
                      child: ListTile(
                        leading: const Icon(Icons.auto_awesome, color: AppColors.primary),
                        title: Text(
                          item['keyword'] ?? '',
                          style: const TextStyle(fontWeight: FontWeight.bold),
                        ),
                        trailing: Row(
                          mainAxisSize: MainAxisSize.min,
                          children: [
                            IconButton(
                              icon: const Icon(Icons.check_circle, color: AppColors.successGreen),
                              onPressed: () => _processKeyword(
                                item['id'] as String,
                                true,
                              ),
                            ),
                            IconButton(
                              icon: const Icon(Icons.cancel, color: AppColors.error),
                              onPressed: () => _processKeyword(
                                item['id'] as String,
                                false,
                              ),
                            ),
                          ],
                        ),
                      ),
                    );
                  },
                ),
        ],
      ),
    );
  }

  Widget _buildUsersTab() {
    return SingleChildScrollView(
      padding: const EdgeInsets.all(20),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const Text(
            'User AI Job Matching Controls',
            style: TextStyle(
              fontSize: 20,
              fontWeight: FontWeight.bold,
              color: AppColors.primary,
            ),
          ),
          const SizedBox(height: 4),
          const Text(
            'Only Master Admin can enable or disable background AI job matching per user account.',
            style: TextStyle(
              fontSize: 13,
              color: AppColors.onSurfaceVariant,
            ),
          ),
          const SizedBox(height: 16),
          _registeredUsers.isEmpty
              ? const Padding(
                  padding: EdgeInsets.all(32),
                  child: Center(
                    child: Text('No registered users found in system.'),
                  ),
                )
              : ListView.builder(
                  shrinkWrap: true,
                  physics: const NeverScrollableScrollPhysics(),
                  itemCount: _registeredUsers.length,
                  itemBuilder: (context, index) {
                    final u = _registeredUsers[index];
                    final String userId = u['id'] as String;
                    final String email = u['primary_email'] as String? ?? 'No email';
                    final String name = u['full_name'] as String? ?? '';
                    final bool aiEnabled = u['ai_matching_enabled'] as bool? ?? false;
                    final bool isAdmin = u['is_master_admin'] as bool? ?? false;

                    return Card(
                      margin: const EdgeInsets.only(bottom: 8),
                      shape: RoundedRectangleBorder(
                        borderRadius: BorderRadius.circular(8),
                        side: const BorderSide(color: AppColors.outlineVariant),
                      ),
                      child: ListTile(
                        leading: CircleAvatar(
                          backgroundColor: isAdmin
                              ? AppColors.primaryContainer
                              : AppColors.surfaceContainerHigh,
                          child: Icon(
                            isAdmin ? Icons.admin_panel_settings : Icons.person,
                            color: isAdmin ? Colors.white : AppColors.outline,
                          ),
                        ),
                        title: Text(
                          name.isNotEmpty ? '$name ($email)' : email,
                          style: const TextStyle(fontWeight: FontWeight.bold),
                        ),
                        subtitle: Text(
                          isAdmin
                              ? 'Role: Master Admin'
                              : 'AI Matching: ${aiEnabled ? "ENABLED" : "DISABLED"}',
                        ),
                        trailing: Switch(
                          value: aiEnabled,
                          onChanged: (val) {
                            _toggleUserAIMatching(userId, val);
                          },
                        ),
                      ),
                    );
                  },
                ),
        ],
      ),
    );
  }
}
