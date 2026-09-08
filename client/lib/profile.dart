import "dart:async";
import "package:flutter/foundation.dart" show kIsWeb;
import "package:flutter/material.dart";
import "package:google_sign_in/google_sign_in.dart";
import "admin.dart";
import "auth.dart";
import "main.dart" show AppColors;
import "preferences.dart" as preferences_page;
import "screens/edit_profile_screen.dart";
import "screens/tailored_documents_screen.dart";
import "services/api_service.dart";
import "services/app_version_service.dart";
import "widgets/tailoring_result_sheet.dart";

void main() {
  WidgetsFlutterBinding.ensureInitialized();
  runApp(const ProfileApp());
}

/// Standalone entry point for the user profile application.
class ProfileApp extends StatelessWidget {
  const ProfileApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: "Professional Profile",
      debugShowCheckedModeBanner: false,
      theme: ThemeData(
        scaffoldBackgroundColor: AppColors.surface,
        fontFamily: "Inter",
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

/// Comprehensive professional profile page rendering user identity, structured background, and tailored documents.
class ProfilePage extends StatefulWidget {
  final Map<String, dynamic>? initialProfileData;
  final Map<String, dynamic>? initialPreferencesData;

  const ProfilePage({
    super.key,
    this.initialProfileData,
    this.initialPreferencesData,
  });

  @override
  State<ProfilePage> createState() => _ProfilePageState();
}

class _ProfilePageState extends State<ProfilePage> {
  preferences_page.PreferenceSummary? _preferenceSummary;
  Map<String, dynamic>? _preferencesData;

  final ApiService _apiService = ApiService();

  Map<String, dynamic>? _userProfile;
  bool _isLoadingProfile = true;
  AppVersionDetails? _appVersionDetails;

  List<TailoredJobDocumentGroup> _tailoredGroups = [];
  bool _isLoadingDocuments = true;
  Timer? _tailoringPollingTimer;

  @override
  void initState() {
    super.initState();
    if (widget.initialProfileData != null) {
      _userProfile = widget.initialProfileData;
      _isLoadingProfile = false;
    } else {
      _loadUserProfile();
    }
    if (widget.initialPreferencesData != null) {
      _preferencesData = widget.initialPreferencesData;
      _applyPreferencesData(widget.initialPreferencesData!);
    } else {
      _loadSavedPreferences();
    }
    _loadAppVersionDetails();
    _loadTailoredDocuments();
  }

  @override
  void dispose() {
    _tailoringPollingTimer?.cancel();
    super.dispose();
  }

  void _checkAndStartTailoringPolling() {
    final hasGenerating = _tailoredGroups.any((group) => group.isGenerating);
    if (hasGenerating && _tailoringPollingTimer == null) {
      _tailoringPollingTimer = Timer.periodic(const Duration(seconds: 4), (_) async {
        final results = await Future.wait([
          _apiService.fetchResumeVersions(),
          _apiService.fetchCoverLetterVersions(),
        ]);
        if (!mounted) return;
        final updatedGroups = groupTailoredDocuments(results[0], results[1]);
        setState(() {
          _tailoredGroups = updatedGroups;
        });
        final stillGenerating = updatedGroups.any((group) => group.isGenerating);
        if (!stillGenerating) {
          _tailoringPollingTimer?.cancel();
          _tailoringPollingTimer = null;
        }
      });
    } else if (!hasGenerating && _tailoringPollingTimer != null) {
      _tailoringPollingTimer?.cancel();
      _tailoringPollingTimer = null;
    }
  }

  Future<void> _loadTailoredDocuments() async {
    final results = await Future.wait([
      _apiService.fetchResumeVersions(),
      _apiService.fetchCoverLetterVersions(),
    ]);
    if (!mounted) return;
    setState(() {
      _tailoredGroups = groupTailoredDocuments(results[0], results[1]);
      _isLoadingDocuments = false;
    });
    _checkAndStartTailoringPolling();
  }

  Future<void> _handleViewTailoredDocument(Map<String, dynamic> doc, String type) async {
    final docId = doc["id"] as String? ?? "";
    final status = doc["status"] as String? ?? "ready";
    final errorMessage = doc["error_message"] as String? ?? "";

    if (status == "generating" || status == "processing") {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(
          content: Text("AI is tailoring this document in the background. It will be ready shortly."),
          backgroundColor: Colors.orange,
        ),
      );
      return;
    }

    if (status == "failed") {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text("Generation failed: ${errorMessage.isNotEmpty ? errorMessage : "Unknown error"}"),
          backgroundColor: Colors.redAccent,
        ),
      );
      return;
    }

    if (docId.isEmpty) return;

    final pdfResponse = type == "resume"
        ? await _apiService.fetchResumeVersionPDF(docId)
        : await _apiService.fetchCoverLetterPDF(docId);

    if (!mounted) return;

    if (pdfResponse != null && pdfResponse["pdf_base64"] != null) {
      await showTailoringResultSheet(
        context: context,
        sessionType: type,
        tailoringResponse: {
          "pdf_base64": pdfResponse["pdf_base64"],
          "pdf_web_url": doc["pdf_url"] ?? "",
          "folder_path": doc["overleaf_folder_path"] ?? "",
          "page_count": pdfResponse["page_count"] ?? doc["page_count"] ?? 1,
        },
      );
    } else {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(
          content: Text("Failed to load PDF from Open-Overleaf."),
          backgroundColor: Colors.redAccent,
        ),
      );
    }
  }

  Future<void> _handleSetDefaultResume(String resumeId) async {
    final success = await _apiService.setDefaultResumeVersion(resumeId);
    if (!mounted) return;
    if (success) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(
          content: Text("Default resume updated."),
          backgroundColor: AppColors.successGreen,
        ),
      );
      _loadTailoredDocuments();
    }
  }

  Future<void> _handleDeleteTailoredDocument(String docId, String type) async {
    final isResume = type == "resume";
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (dialogContext) => AlertDialog(
        title: Text(isResume ? "Delete Resume Version" : "Delete Cover Letter"),
        content: const Text(
          "Remove this version reference from Job Cruiser? The files will remain preserved in your Open-Overleaf project.",
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(dialogContext).pop(false),
            child: const Text("Cancel"),
          ),
          TextButton(
            onPressed: () => Navigator.of(dialogContext).pop(true),
            style: TextButton.styleFrom(foregroundColor: AppColors.error),
            child: const Text("Delete"),
          ),
        ],
      ),
    );

    if (confirmed != true) return;

    final success = isResume
        ? await _apiService.deleteResumeVersion(docId)
        : await _apiService.deleteCoverLetterVersion(docId);

    if (!mounted) return;

    if (success) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text("${isResume ? "Resume" : "Cover letter"} version removed.")),
      );
      _loadTailoredDocuments();
    }
  }

  Future<void> _loadAppVersionDetails() async {
    const service = AppVersionService();
    final details = await service.getVersionDetails();
    if (!mounted) return;
    setState(() => _appVersionDetails = details);
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

  void _applyPreferencesData(Map<String, dynamic> apiPref) {
    if ((apiPref["target_roles"] as List? ?? []).isNotEmpty) {
      final industries = List<String>.from(apiPref["target_industries"] as List? ?? ["Tech"]);
      final targetRoles = List<String>.from(apiPref["target_roles"] as List? ?? []);
      final double minSalary = ((apiPref["min_salary"] as num? ?? 0).toDouble() / 1000);
      final bool aiMatchingEnabled = apiPref["ai_matching_enabled"] as bool? ?? false;

      _preferenceSummary = preferences_page.PreferenceSummary(
        industries: industries,
        targetRoles: targetRoles,
        baseSalary: minSalary,
        equityExpectation: aiMatchingEnabled ? "AI Matching: Enabled (Managed by Admin)" : "AI Matching: Disabled (Managed by Admin)",
      );
    }
  }

  Future<void> _loadSavedPreferences() async {
    final apiPref = await _apiService.fetchPreferences();
    if (!mounted) return;

    setState(() {
      _preferencesData = apiPref;
      if (apiPref != null) {
        _applyPreferencesData(apiPref);
      }
    });

    if (apiPref == null || (apiPref["target_roles"] as List? ?? []).isEmpty) {
      final savedPreferences = await preferences_page.PreferenceSummary.load();
      if (mounted && savedPreferences != null) {
        setState(() {
          _preferenceSummary = savedPreferences;
        });
      }
    }
  }

  Future<void> _openEditProfile(BuildContext context) async {
    final updated = await Navigator.of(context).push<bool>(
      MaterialPageRoute(
        builder: (_) => EditProfileScreen(
          initialProfileData: _preferencesData,
        ),
      ),
    );

    if (updated == true && mounted) {
      await _loadSavedPreferences();
      await _loadUserProfile();
    }
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

  @override
  Widget build(BuildContext context) {
    final isDesktop = MediaQuery.of(context).size.width >= 960;
    return Scaffold(
      appBar: isDesktop ? null : _buildAppBar(),
      body: SingleChildScrollView(
        padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 24),
        child: Center(
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: 768),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                _buildProfileBento(context),
                const SizedBox(height: 24),
                _buildSectionTitle("JOB PREFERENCES & TARGETS"),
                const SizedBox(height: 8),
                _buildPreferencesSection(context),
                const SizedBox(height: 24),
                _buildSectionTitle("DOCUMENTS & TAILORING"),
                const SizedBox(height: 8),
                _buildDocumentsSection(),
                const SizedBox(height: 24),
                _buildSectionTitle("ACCOUNT & SECURITY"),
                const SizedBox(height: 8),
                _buildSecuritySection(),
                const SizedBox(height: 24),
                _buildSectionTitle("APP INFORMATION"),
                const SizedBox(height: 8),
                _buildAppInfoSection(),
                const SizedBox(height: 24),
                _buildVersionFooter(),
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
            "Professional Profile",
            style: TextStyle(
              color: AppColors.primary,
              fontSize: 20,
              fontWeight: FontWeight.w700,
              fontFamily: "Inter",
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
    final String fullName = _preferencesData?["full_name"]?.toString().isNotEmpty == true
        ? _preferencesData!["full_name"].toString()
        : _userProfile?["full_name"] ?? "User";
    final String? avatarUrl = _userProfile?["avatar_url"];
    final String primaryEmail = _preferencesData?["email"]?.toString().isNotEmpty == true
        ? _preferencesData!["email"].toString()
        : _userProfile?["primary_email"] ?? "No email provided";

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
            children: [
              Container(
                width: 72,
                height: 72,
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
                          fullName.isNotEmpty ? fullName[0].toUpperCase() : "?",
                          style: const TextStyle(
                            fontSize: 28,
                            fontWeight: FontWeight.w600,
                            color: AppColors.outline,
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
                        fontWeight: FontWeight.w700,
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
                            "Identity Verified",
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
          const SizedBox(height: 20),
          const Divider(height: 1, color: AppColors.outlineVariant),
          const SizedBox(height: 16),
          Row(
            children: [
              Expanded(
                child: ElevatedButton.icon(
                  onPressed: () => _openEditProfile(context),
                  icon: const Icon(Icons.edit_outlined, size: 16),
                  label: const Text("Edit Profile & Bio"),
                  style: ElevatedButton.styleFrom(
                    backgroundColor: AppColors.primary,
                    foregroundColor: Colors.white,
                    padding: const EdgeInsets.symmetric(vertical: 12),
                  ),
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: OutlinedButton.icon(
                  onPressed: () => _openPreferences(context),
                  icon: const Icon(Icons.tune, size: 16),
                  label: const Text("Job Preferences"),
                  style: OutlinedButton.styleFrom(
                    foregroundColor: AppColors.primary,
                    side: const BorderSide(color: AppColors.outlineVariant),
                    padding: const EdgeInsets.symmetric(vertical: 12),
                  ),
                ),
              ),
            ],
          ),
        ],
      ),
    );
  }

  Widget _buildPreferencesSection(BuildContext context) {
    final summary = _preferenceSummary;
    final String rolesText = summary != null && summary.targetRoles.isNotEmpty
        ? summary.targetRoles.join(", ")
        : "Any Role";
    final String industriesText = summary != null && summary.industries.isNotEmpty
        ? summary.industries.join(", ")
        : "Any Industry";
    final String salaryText = summary != null && summary.baseSalary > 0
        ? (summary.baseSalary <= 100 ? "₹${summary.baseSalary.toInt()} LPA+" : "\$${summary.baseSalary.toInt()}k+")
        : "Any Salary Target";

    final rawLocations = _preferencesData?["target_locations"] as List<dynamic>? ?? [];
    final locationsText = rawLocations.isNotEmpty ? rawLocations.join(", ") : "Any Location";

    final rawWorkModels = _preferencesData?["work_models"] as List<dynamic>? ?? [];
    final workModelsText = rawWorkModels.isNotEmpty ? rawWorkModels.join(", ") : "Any Work Model";

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
                "Matching Setup",
                style: TextStyle(
                  fontSize: 18,
                  fontWeight: FontWeight.bold,
                  color: AppColors.primary,
                ),
              ),
              OutlinedButton.icon(
                onPressed: () => _openPreferences(context),
                icon: const Icon(Icons.edit, size: 16),
                label: const Text("Edit Preferences"),
                style: OutlinedButton.styleFrom(
                  foregroundColor: AppColors.primary,
                  side: const BorderSide(color: AppColors.outlineVariant),
                ),
              ),
            ],
          ),
          const SizedBox(height: 16),
          _buildPrefRowItem("Target Roles", rolesText),
          const Divider(height: 24, color: AppColors.outlineVariant),
          _buildPrefRowItem("Preferred Industries", industriesText),
          const Divider(height: 24, color: AppColors.outlineVariant),
          _buildPrefRowItem("Target Locations", locationsText),
          const Divider(height: 24, color: AppColors.outlineVariant),
          _buildPrefRowItem("Work Models", workModelsText),
          const Divider(height: 24, color: AppColors.outlineVariant),
          _buildPrefRowItem("Base Salary Target", salaryText),
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

  Widget _buildDocumentsSection() {
    if (_isLoadingDocuments) {
      return Container(
        padding: const EdgeInsets.all(24),
        decoration: BoxDecoration(
          color: AppColors.surfaceContainerLowest,
          borderRadius: BorderRadius.circular(12),
          border: Border.all(color: AppColors.outlineVariant),
        ),
        child: const Center(
          child: CircularProgressIndicator(),
        ),
      );
    }

    if (_tailoredGroups.isEmpty) {
      return Container(
        padding: const EdgeInsets.all(24),
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
            Container(
              padding: const EdgeInsets.all(16),
              decoration: const BoxDecoration(
                color: AppColors.surfaceContainerHigh,
                shape: BoxShape.circle,
              ),
              child: const Icon(
                Icons.folder_shared_outlined,
                size: 32,
                color: AppColors.outline,
              ),
            ),
            const SizedBox(height: 12),
            const Text(
              "No Tailored Documents Yet",
              style: TextStyle(
                fontSize: 16,
                fontWeight: FontWeight.w700,
                color: AppColors.primary,
              ),
            ),
            const SizedBox(height: 6),
            const Text(
              "Tap \"Tailor Application\" on any job in your feed to generate ATS-tailored CV and Cover Letter together.",
              textAlign: TextAlign.center,
              style: TextStyle(
                fontSize: 13,
                color: AppColors.onSurfaceVariant,
              ),
            ),
          ],
        ),
      );
    }

    return Column(
      children: [
        ..._tailoredGroups.map((group) {
          return TailoredJobCard(
            group: group,
            onViewDocument: _handleViewTailoredDocument,
            onDeleteDocument: _handleDeleteTailoredDocument,
            onSetDefaultResume: _handleSetDefaultResume,
          );
        }),
        const SizedBox(height: 4),
        Center(
          child: TextButton.icon(
            onPressed: () {
              Navigator.of(context).push(
                MaterialPageRoute(
                  builder: (_) => const TailoredDocumentsScreen(),
                ),
              );
            },
            icon: const Icon(Icons.open_in_new, size: 16, color: AppColors.secondary),
            label: const Text(
              "Open Full-Screen Documents Manager",
              style: TextStyle(
                fontSize: 12,
                fontWeight: FontWeight.w600,
                color: AppColors.secondary,
              ),
            ),
          ),
        ),
      ],
    );
  }

  Widget _buildSecuritySection() {
    final bool isMasterAdmin = _userProfile?["is_master_admin"] as bool? ?? false;

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
              title: "Master Admin Control Panel",
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
            title: "Sign Out",
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

  Widget _buildAppInfoSection() {
    final versionText = _appVersionDetails?.compactVersion ?? "v1.0.0+1";
    final platformText = _appVersionDetails?.platformName ?? (kIsWeb ? "Web" : "Mobile");
    final buildText = _appVersionDetails != null ? "Build ${_appVersionDetails!.buildNumber}" : "Build 1";

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
          _buildInfoRow(
            icon: Icons.info_outline,
            label: "Application Version",
            value: versionText,
            hasBorder: true,
          ),
          _buildInfoRow(
            icon: kIsWeb ? Icons.language : Icons.devices,
            label: "Platform & Environment",
            value: platformText,
            hasBorder: true,
          ),
          _buildInfoRow(
            icon: Icons.tag,
            label: "Build Number",
            value: buildText,
            hasBorder: false,
          ),
        ],
      ),
    );
  }

  Widget _buildInfoRow({
    required IconData icon,
    required String label,
    required String value,
    required bool hasBorder,
  }) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 14),
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
              Icon(icon, color: AppColors.primary, size: 20),
              const SizedBox(width: 12),
              Text(
                label,
                style: const TextStyle(fontSize: 14, color: AppColors.primary),
              ),
            ],
          ),
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
            decoration: BoxDecoration(
              color: AppColors.surfaceContainerHigh,
              borderRadius: BorderRadius.circular(6),
            ),
            child: Text(
              value,
              style: const TextStyle(
                fontSize: 12,
                fontWeight: FontWeight.w600,
                color: AppColors.onSurface,
              ),
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildVersionFooter() {
    final displayText = _appVersionDetails?.displayVersion ?? "v1.0.0 • ${kIsWeb ? "Web" : "Mobile"}";
    return Center(
      child: Padding(
        padding: const EdgeInsets.only(bottom: 16),
        child: Text(
          "Job Cruiser $displayText",
          style: const TextStyle(
            fontSize: 12,
            fontWeight: FontWeight.w500,
            color: AppColors.outline,
          ),
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
          letterSpacing: 0.6,
          color: AppColors.onSurfaceVariant,
        ),
      ),
    );
  }
}
