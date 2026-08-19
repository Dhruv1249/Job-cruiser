import 'package:flutter/material.dart';
import 'services/api_service.dart';
import 'package:google_sign_in/google_sign_in.dart';
import 'preferences.dart' as preferences_page;
import 'main.dart' show AppColors;
import 'auth.dart';
import 'admin.dart';
import 'screens/resume_versions_screen.dart';
import 'screens/cover_letters_screen.dart';

void main() {
  WidgetsFlutterBinding.ensureInitialized();
  runApp(const ProfileApp());
}

class ProfileApp extends StatelessWidget {
  const ProfileApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Professional Profile',
      debugShowCheckedModeBanner: false,
      theme: ThemeData(
        scaffoldBackgroundColor: AppColors.surface,
        fontFamily: 'Inter',
        colorScheme: ColorScheme.fromSeed(
          seedColor: AppColors.primary,
          surface: AppColors.surface,
          primary: AppColors.primary,
          error: AppColors.error,
        ),
      ),
      home: const ProfilePage(),
    );
  }
}

class ProfilePage extends StatefulWidget {
  const ProfilePage({super.key});

  @override
  State<ProfilePage> createState() => _ProfilePageState();
}

class _ProfilePageState extends State<ProfilePage> {
  preferences_page.PreferenceSummary? _preferenceSummary;

  final ApiService _apiService = ApiService();

  Map<String, dynamic>? _userProfile;
  bool _isLoadingProfile = true;

  @override
  void initState() {
    super.initState();
    _loadSavedPreferences();
    _loadUserProfile();
  }

  Future<void> _loadUserProfile() async {
    final profile = await _apiService.fetchProfile();
    if (!mounted) {
      return;
    }

    setState(() {
      _userProfile = profile;
      _isLoadingProfile = false;
    });
  }

  Future<void> _loadSavedPreferences() async {
    final apiPref = await _apiService.fetchPreferences();
    if (!mounted) return;

    if (apiPref != null && (apiPref['target_roles'] as List? ?? []).isNotEmpty) {
      final industries = List<String>.from(apiPref['target_industries'] as List? ?? ['Tech']);
      final targetRoles = List<String>.from(apiPref['target_roles'] as List? ?? []);
      final double minSalary = ((apiPref['min_salary'] as num? ?? 0).toDouble() / 1000);
      final bool aiMatchingEnabled = apiPref['ai_matching_enabled'] as bool? ?? false;

      setState(() {
        _preferenceSummary = preferences_page.PreferenceSummary(
          industries: industries,
          targetRoles: targetRoles,
          baseSalary: minSalary,
          equityExpectation: aiMatchingEnabled ? 'AI Matching: Enabled (Managed by Admin)' : 'AI Matching: Disabled (Managed by Admin)',
        );
      });
    } else {
      final savedPreferences = await preferences_page.PreferenceSummary.load();
      if (mounted && savedPreferences != null) {
        setState(() {
          _preferenceSummary = savedPreferences;
        });
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: _buildAppBar(),
      body: SingleChildScrollView(
        padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 24),
        child: Center(
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: 768), // max-w-3xl
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                _buildProfileBento(context),
                const SizedBox(height: 24),
                _buildSectionTitle('JOB PREFERENCES & TARGETS'),
                const SizedBox(height: 8),
                _buildPreferencesSection(context),
                const SizedBox(height: 24),
                _buildSectionTitle('DOCUMENTS & TAILORING'),
                const SizedBox(height: 8),
                _buildDocumentsSection(),
                const SizedBox(height: 24),
                _buildSectionTitle('ACCOUNT & SECURITY'),
                const SizedBox(height: 8),
                _buildSecuritySection(),
              ],
            ),
          ),
        ),
      ),
    );
  }

  PreferredSizeWidget _buildAppBar() {
    return AppBar(
      backgroundColor: AppColors.surface,
      elevation: 0,
      scrolledUnderElevation: 0,
      bottom: PreferredSize(
        preferredSize: const Size.fromHeight(1),
        child: Container(color: AppColors.outlineVariant, height: 1),
      ),
      titleSpacing: 20,
      title: Row(
        children: [
          Container(
            width: 40,
            height: 40,
            decoration: BoxDecoration(
              color: AppColors.surfaceContainerHigh,
              shape: BoxShape.circle,
              border: Border.all(color: AppColors.outlineVariant),
            ),
            child: const Icon(
              Icons.person_outline,
              color: AppColors.outline,
              size: 24,
            ),
          ),
          const SizedBox(width: 12),
          const Text(
            'Professional Profile',
            style: TextStyle(
              color: AppColors.primary,
              fontSize: 20,
              fontWeight: FontWeight.w700,
              fontFamily: 'Inter',
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildProfileBento(BuildContext context) {
    if (_isLoadingProfile) {
      return const Center(
        child: Padding(
          padding: EdgeInsets.all(32),
          child: CircularProgressIndicator(),
        ),
      );
    }
    final String fullName = _userProfile?['full_name'] ?? 'Guest User';
    final String? avatarUrl = _userProfile?['avatar_url'];
    final String primaryEmail =
        _userProfile?['primary_email'] ?? 'No email provided';

    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: AppColors.surfaceContainerLowest,
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: AppColors.outlineVariant),
        boxShadow: [
          BoxShadow(
            color: Colors.black.withValues(alpha: 0.02),
            blurRadius: 4,
            offset: const Offset(0, 2),
          ),
        ],
      ),
      child: Row(
        children: [
          Container(
            width: 80,
            height: 80,
            decoration: BoxDecoration(
              color: AppColors.surfaceContainer,
              shape: BoxShape.circle,
              image: avatarUrl != null
                  ? DecorationImage(
                      image: NetworkImage(avatarUrl),
                      fit: BoxFit.cover,
                    )
                  : null,
            ),
            child: avatarUrl == null
                ? Center(
                    child: Text(
                      fullName.isNotEmpty ? fullName[0].toUpperCase() : '?',
                      style: const TextStyle(
                        fontSize: 32,
                        fontWeight: FontWeight.w600,
                        color: AppColors
                            .outline, // Matches your fallback icon color
                      ),
                    ),
                  )
                : null,
          ),
          const SizedBox(width: 16),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  fullName,
                  style: const TextStyle(
                    fontSize: 20,
                    fontWeight: FontWeight.w600,
                    color: AppColors.primary,
                  ),
                ),
                const SizedBox(height: 4),
                Text(
                  primaryEmail,
                  style: const TextStyle(
                    fontSize: 14,
                    color: AppColors.onSurfaceVariant,
                  ),
                ),
                const SizedBox(height: 8),
                Container(
                  padding: const EdgeInsets.symmetric(
                    horizontal: 8,
                    vertical: 4,
                  ),
                  decoration: BoxDecoration(
                    color: AppColors.tertiaryFixedDim.withValues(alpha: 0.2),
                    border: Border.all(color: AppColors.tertiaryFixedDim),
                    borderRadius: BorderRadius.circular(16),
                  ),
                  child: const Row(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      Icon(
                        Icons.verified,
                        color: AppColors.onTertiaryContainer,
                        size: 14,
                      ),
                      SizedBox(width: 4),
                      Text(
                        'Identity Verified',
                        style: TextStyle(
                          fontSize: 11,
                          fontWeight: FontWeight.w600,
                          color: AppColors.onTertiaryContainer,
                        ),
                      ),
                    ],
                  ),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildPreferencesSection(BuildContext context) {
    final summary = _preferenceSummary;
    final String rolesText = summary != null && summary.targetRoles.isNotEmpty
        ? summary.targetRoles.join(', ')
        : 'Any Role';
    final String industriesText = summary != null && summary.industries.isNotEmpty
        ? summary.industries.join(', ')
        : 'Any Industry';
    final String salaryText = summary != null && summary.baseSalary > 0
        ? (summary.baseSalary <= 100 ? '₹${summary.baseSalary.toInt()} LPA+' : '\$${summary.baseSalary.toInt()}k+')
        : 'Any Salary Target';

    return Container(
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: AppColors.surfaceContainerLowest,
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: AppColors.outlineVariant),
        boxShadow: [
          BoxShadow(
            color: Colors.black.withValues(alpha: 0.02),
            blurRadius: 4,
            offset: const Offset(0, 2),
          ),
        ],
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            mainAxisAlignment: MainAxisAlignment.spaceBetween,
            children: [
              const Text(
                'Matching Setup',
                style: TextStyle(
                  fontSize: 18,
                  fontWeight: FontWeight.bold,
                  color: AppColors.primary,
                ),
              ),
              OutlinedButton.icon(
                onPressed: () => _openPreferences(context),
                icon: const Icon(Icons.edit, size: 16),
                label: const Text('Edit Preferences'),
                style: OutlinedButton.styleFrom(
                  foregroundColor: AppColors.primary,
                  side: const BorderSide(color: AppColors.outlineVariant),
                ),
              ),
            ],
          ),
          const SizedBox(height: 16),
          _buildPrefRowItem('Target Roles', rolesText),
          const Divider(height: 24, color: AppColors.outlineVariant),
          _buildPrefRowItem('Preferred Industries', industriesText),
          const Divider(height: 24, color: AppColors.outlineVariant),
          _buildPrefRowItem('Base Salary Target', salaryText),
        ],
      ),
    );
  }

  Widget _buildPrefRowItem(String label, String value) {
    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        SizedBox(
          width: 140,
          child: Text(
            label,
            style: const TextStyle(
              fontSize: 14,
              color: AppColors.onSurfaceVariant,
              fontWeight: FontWeight.w500,
            ),
          ),
        ),
        Expanded(
          child: Text(
            value,
            style: const TextStyle(
              fontSize: 14,
              color: AppColors.primary,
              fontWeight: FontWeight.w600,
            ),
          ),
        ),
      ],
    );
  }

  Future<void> _openPreferences(BuildContext context) async {
    await Navigator.of(context).push(
      MaterialPageRoute(
        builder: (_) => preferences_page.SetPreferencesScreen(
          initialPreferences: _preferenceSummary,
        ),
      ),
    );

    if (!mounted) return;
    await _loadSavedPreferences();
    await _loadUserProfile();
  }

  Widget _buildDocumentsSection() {
    return Container(
      decoration: BoxDecoration(
        color: AppColors.surfaceContainerLowest,
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: AppColors.outlineVariant),
        boxShadow: [
          BoxShadow(
            color: Colors.black.withValues(alpha: 0.02),
            blurRadius: 4,
            offset: const Offset(0, 2),
          ),
        ],
      ),
      child: Column(
        children: [
          _buildActionTile(
            icon: Icons.description_outlined,
            title: 'Tailored Resumes',
            hasBorder: true,
            onTap: () {
              Navigator.of(context).push(
                MaterialPageRoute(
                  builder: (_) => const ResumeVersionsScreen(),
                ),
              );
            },
          ),
          _buildActionTile(
            icon: Icons.mail_outline,
            title: 'Generated Cover Letters',
            hasBorder: false,
            onTap: () {
              Navigator.of(context).push(
                MaterialPageRoute(
                  builder: (_) => const CoverLettersScreen(),
                ),
              );
            },
          ),
        ],
      ),
    );
  }



  Widget _buildSecuritySection() {
    final bool isMasterAdmin = _userProfile?['is_master_admin'] as bool? ?? false;

    return Container(
      decoration: BoxDecoration(
        color: AppColors.surfaceContainerLowest,
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: AppColors.outlineVariant),
        boxShadow: [
          BoxShadow(
            color: Colors.black.withValues(alpha: 0.02),
            blurRadius: 4,
            offset: const Offset(0, 2),
          ),
        ],
      ),
      child: Column(
        children: [
          if (isMasterAdmin)
            _buildActionTile(
              icon: Icons.admin_panel_settings,
              title: 'Master Admin Control Panel',
              hasBorder: true,
              onTap: () {
                Navigator.of(context).push(
                  MaterialPageRoute(
                    builder: (_) => const MasterAdminScreen(),
                  ),
                );
              },
            ),
          _buildActionTile(
            icon: Icons.logout,
            title: 'Sign Out',
            hasBorder: false,
            isDestructive: true,
            onTap: () async {
              await _apiService.clearToken();
              try {
                await GoogleSignIn().signOut();
              } catch (_) {}
              if (!mounted) return;
              Navigator.of(context).pushAndRemoveUntil(
                MaterialPageRoute(builder: (_) => const AuthScreen()),
                (route) => false,
              );
            },
          ),
        ],
      ),
    );
  }

  Widget _buildActionTile({
    required IconData icon,
    required String title,
    required bool hasBorder,
    bool isDestructive = false,
    VoidCallback? onTap,
  }) {
    final color = isDestructive ? AppColors.error : AppColors.primary;

    return InkWell(
      onTap: onTap,
      child: Container(
        padding: const EdgeInsets.all(16),
        decoration: BoxDecoration(
          border: hasBorder
              ? Border(
                  bottom: BorderSide(
                    color: AppColors.outlineVariant.withValues(alpha: 0.5),
                  ),
                )
              : null,
        ),
        child: Row(
          mainAxisAlignment: MainAxisAlignment.spaceBetween,
          children: [
            Row(
              children: [
                Icon(icon, color: color, size: 24),
                const SizedBox(width: 12),
                Text(title, style: TextStyle(fontSize: 14, color: color)),
              ],
            ),
            if (!isDestructive)
              const Icon(Icons.chevron_right, color: AppColors.outline),
          ],
        ),
      ),
    );
  }

  Widget _buildSectionTitle(String title) {
    return Padding(
      padding: const EdgeInsets.only(left: 4),
      child: Text(
        title,
        style: const TextStyle(
          fontSize: 12,
          fontWeight: FontWeight.w500,
          letterSpacing: 0.6, // tracking-widest approximation
          color: AppColors.onSurfaceVariant,
        ),
      ),
    );
  }
}

