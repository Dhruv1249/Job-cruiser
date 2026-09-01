import 'dart:async';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'details.dart' as details_page;
import 'profile.dart';
import 'preferences.dart' as preferences_page;
import 'auth.dart';
import 'onboarding.dart';
import 'tracker.dart';
import 'models/job.dart';
import 'models/job_filter_state.dart';
import 'services/api_service.dart';
import 'services/notification_service.dart';
import 'services/update_checker_service.dart';
import 'widgets/company_logo_avatar.dart';
import 'widgets/notifications_sheet.dart';
import 'widgets/job_filter_bar.dart';
import 'widgets/job_filter_dialog.dart';
import 'widgets/job_detail_panel.dart';
import 'widgets/update_banner.dart';
import 'package:flutter_dotenv/flutter_dotenv.dart';

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();
  await dotenv.load(fileName: ".env");
  await NotificationService.instance.initialize();
  runApp(const MyApp());
}

/// Palette constants defining theme styling across the application.
class AppColors {
  static const Color background = Color(0xFFF7F9FB);
  static const Color surface = Color(0xFFF7F9FB);
  static const Color surfaceContainerLowest = Color(0xFFFFFFFF);
  static const Color surfaceContainer = Color(0xFFECEEF0);
  static const Color surfaceContainerHigh = Color(0xFFE6E8EA);
  static const Color surfaceContainerLow = Color(0xFFF2F4F6);
  static const Color primary = Color(0xFF000000);
  static const Color primaryContainer = Color(0xFF131B2E);
  static const Color onPrimaryContainer = Color(0xFF7C839B);
  static const Color onSurfaceVariant = Color(0xFF45464D);
  static const Color onSurface = Color(0xFF191C1E);
  static const Color outline = Color(0xFF76777D);
  static const Color outlineVariant = Color(0xFFC6C6CD);
  static const Color secondary = Color(0xFF515F74);
  static const Color secondaryContainer = Color(0xFFD5E3FD);
  static const Color onSecondaryContainer = Color(0xFF57657B);
  static const Color surfaceDim = Color(0xFFD8DADC);
  static const Color surfaceTint = Color(0xFF565E74);
  static const Color error = Color(0xFFBA1A1A);
  static const Color tertiaryFixed = Color(0xFF6FFBBE);
  static const Color tertiaryFixedDim = Color(0xFF4EDEA3);
  static const Color onTertiaryContainer = Color(0xFF009668);
  static const Color onTertiary = Color(0xFFFFFFFF);
  static const Color matchGreen = Color(0xFF10B981);
  static const Color inverseSurface = Color(0xFF2D3133);
  static const Color successGreen = Color(0xFF10B981);
  static const Color slate900 = Color(0xFF0F172A);
  static const Color sliderInactive = Color(0xFFE2E8F0);
}

/// Main entry widget configuring theme and initial routing.
class MyApp extends StatelessWidget {
  const MyApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      debugShowCheckedModeBanner: false,
      title: 'Job Cruiser',
      theme: ThemeData(
        useMaterial3: true,
        scaffoldBackgroundColor: AppColors.surface,
        fontFamily: 'Inter',
        colorScheme: ColorScheme.fromSeed(
          seedColor: AppColors.primary,
          surface: AppColors.surface,
          primary: AppColors.primary,
          onSurfaceVariant: AppColors.onSurfaceVariant,
        ),
        navigationBarTheme: NavigationBarThemeData(
          backgroundColor: AppColors.surface,
          indicatorColor: AppColors.secondaryContainer,
          labelTextStyle: WidgetStateProperty.resolveWith((states) {
            if (states.contains(WidgetState.selected)) {
              return const TextStyle(
                fontSize: 11,
                fontWeight: FontWeight.w600,
                color: AppColors.onSecondaryContainer,
              );
            }
            return const TextStyle(
              fontSize: 11,
              fontWeight: FontWeight.w600,
              color: AppColors.onSurfaceVariant,
            );
          }),
          iconTheme: WidgetStateProperty.resolveWith((states) {
            if (states.contains(WidgetState.selected)) {
              return const IconThemeData(
                color: AppColors.onSecondaryContainer,
              );
            }
            return const IconThemeData(
              color: AppColors.onSurfaceVariant,
            );
          }),
        ),
      ),
      home: const AppInitializer(),
    );
  }
}

/// Widget verifying credentials and routing to onboarding or authenticated shell.
class AppInitializer extends StatefulWidget {
  const AppInitializer({super.key});

  @override
  State<AppInitializer> createState() => _AppInitializerState();
}

class _AppInitializerState extends State<AppInitializer> {
  final ApiService _apiService = ApiService();
  bool _isLoading = true;
  Widget _homeScreen = const AuthScreen();

  @override
  void initState() {
    super.initState();
    _checkAuthAndPreferences();
  }

  Future<void> _checkAuthAndPreferences() async {
    final token = await _apiService.getToken();
    if (token == null || token.isEmpty) {
      if (mounted) {
        setState(() {
          _homeScreen = const AuthScreen();
          _isLoading = false;
        });
      }
      return;
    }

    final profile = await _apiService.fetchProfile();
    if (!mounted) return;

    if (profile == null) {
      setState(() {
        _homeScreen = const AuthScreen();
        _isLoading = false;
      });
    } else {
      final bool hasPreferences = profile['has_preferences'] as bool? ?? false;
      setState(() {
        _homeScreen = hasPreferences ? const JobCruiserShell() : const OnboardingWizardScreen();
        _isLoading = false;
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    if (_isLoading) {
      return const Scaffold(
        body: Center(child: CircularProgressIndicator()),
      );
    }
    return _homeScreen;
  }
}

/// Main application shell providing responsive desktop header and mobile bottom navigation.
class JobCruiserShell extends StatefulWidget {
  const JobCruiserShell({super.key});

  @override
  State<JobCruiserShell> createState() => _JobCruiserShellState();
}

class _JobCruiserShellState extends State<JobCruiserShell> {
  int _currentIndex = 0;
  final ValueNotifier<int> _inboxRefreshTrigger = ValueNotifier<int>(0);

  @override
  void dispose() {
    _inboxRefreshTrigger.dispose();
    super.dispose();
  }

  void _openJobDetails(MatchedJob job) async {
    await Navigator.of(context).push(
      MaterialPageRoute(
        builder: (_) => details_page.CompanyDetailsPage(job: job),
      ),
    );
    setState(() {});
  }

  @override
  Widget build(BuildContext context) {
    final screenWidth = MediaQuery.of(context).size.width;
    final isDesktop = screenWidth >= 960;

    return Scaffold(
      appBar: isDesktop ? _buildDesktopTopNav() : null,
      body: IndexedStack(
        index: _currentIndex,
        children: [
          MyHomePage(
            onSelectJob: _openJobDetails,
            refreshTrigger: _inboxRefreshTrigger,
          ),
          const ApplicationTrackerPage(),
          const ProfilePage(),
        ],
      ),
      bottomNavigationBar: isDesktop ? null : _buildMobileBottomNav(),
    );
  }

  PreferredSizeWidget _buildDesktopTopNav() {
    return AppBar(
      backgroundColor: AppColors.surfaceContainerLowest,
      elevation: 0,
      scrolledUnderElevation: 0,
      bottom: PreferredSize(
        preferredSize: const Size.fromHeight(1.0),
        child: Container(
          color: AppColors.outlineVariant.withValues(alpha: 0.5),
          height: 1.0,
        ),
      ),
      titleSpacing: 24,
      title: Row(
        children: [
          Container(
            width: 32,
            height: 32,
            decoration: BoxDecoration(
              borderRadius: BorderRadius.circular(8),
              color: AppColors.primary,
            ),
            child: const Icon(
              Icons.auto_awesome,
              size: 18,
              color: Colors.white,
            ),
          ),
          const SizedBox(width: 12),
          const Text(
            'Job Cruiser',
            style: TextStyle(
              fontSize: 18,
              fontWeight: FontWeight.w800,
              color: AppColors.primary,
              letterSpacing: -0.3,
            ),
          ),
          const SizedBox(width: 32),
          _buildDesktopNavItem(0, 'Inbox & Matches', Icons.chat_bubble_outline, Icons.chat_bubble),
          const SizedBox(width: 8),
          _buildDesktopNavItem(1, 'CRM Tracker', Icons.work_history_outlined, Icons.work_history),
          const SizedBox(width: 8),
          _buildDesktopNavItem(2, 'Profile & Preferences', Icons.account_circle_outlined, Icons.account_circle),
        ],
      ),
    );
  }

  Widget _buildDesktopNavItem(int index, String label, IconData unselectedIcon, IconData selectedIcon) {
    final isSelected = _currentIndex == index;
    return InkWell(
      onTap: () {
        if (_currentIndex == index && index == 0) {
          _inboxRefreshTrigger.value++;
        } else {
          setState(() => _currentIndex = index);
        }
      },
      borderRadius: BorderRadius.circular(8),
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 8),
        decoration: BoxDecoration(
          color: isSelected ? AppColors.surfaceContainerHigh : Colors.transparent,
          borderRadius: BorderRadius.circular(8),
        ),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(
              isSelected ? selectedIcon : unselectedIcon,
              size: 16,
              color: isSelected ? AppColors.primary : AppColors.onSurfaceVariant,
            ),
            const SizedBox(width: 8),
            Text(
              label,
              style: TextStyle(
                fontSize: 13,
                fontWeight: isSelected ? FontWeight.w700 : FontWeight.w500,
                color: isSelected ? AppColors.primary : AppColors.onSurfaceVariant,
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildMobileBottomNav() {
    return Container(
      decoration: BoxDecoration(
        border: const Border(
          top: BorderSide(color: AppColors.outlineVariant, width: 1.0),
        ),
        boxShadow: [
          BoxShadow(
            color: Colors.black.withValues(alpha: 0.05),
            blurRadius: 12,
            offset: const Offset(0, -4),
          )
        ],
      ),
      child: NavigationBar(
        selectedIndex: _currentIndex,
        onDestinationSelected: (int index) {
          if (_currentIndex == index && index == 0) {
            _inboxRefreshTrigger.value++;
          } else {
            setState(() {
              _currentIndex = index;
            });
          }
        },
        destinations: const [
          NavigationDestination(
            icon: Icon(Icons.chat_bubble_outline),
            selectedIcon: Icon(Icons.chat_bubble),
            label: 'Inbox',
          ),
          NavigationDestination(
            icon: Icon(Icons.work_history_outlined),
            selectedIcon: Icon(Icons.work_history),
            label: 'Tracker',
          ),
          NavigationDestination(
            icon: Icon(Icons.account_circle_outlined),
            selectedIcon: Icon(Icons.account_circle),
            label: 'Profile',
          ),
        ],
      ),
    );
  }
}

/// Primary feed page rendering job opportunities with responsive master-detail layouts.
class MyHomePage extends StatefulWidget {
  const MyHomePage({
    super.key,
    required this.onSelectJob,
    this.refreshTrigger,
  });

  final Function(MatchedJob job) onSelectJob;
  final ValueListenable<int>? refreshTrigger;

  @override
  State<MyHomePage> createState() => _MyHomePageState();
}

class _MyHomePageState extends State<MyHomePage> {
  final ApiService _apiService = ApiService();
  final UpdateCheckerService _updateCheckerService = UpdateCheckerService();
  final TextEditingController _searchController = TextEditingController();
  final ScrollController _scrollController = ScrollController();

  JobFilterState _filterState = const JobFilterState();
  List<MatchedJob> _matchedJobs = [];
  MatchedJob? _selectedJob;
  bool _isLoading = true;
  bool _isFetchingMore = false;
  bool _hasMore = true;
  bool _isMatchEngineRunning = false;
  int _pendingMatchCount = 0;
  int _unreadNotificationCount = 0;
  Timer? _notificationPollingTimer;
  Timer? _searchDebounceTimer;
  PendingUpdate? _pendingUpdate;
  bool _updateBannerDismissed = false;
  final Set<String> _seenNotificationIds = {};
  bool _hasInitialNotificationSync = false;

  @override
  void initState() {
    super.initState();
    widget.refreshTrigger?.addListener(_scrollToTopAndRefresh);
    _initializeFilterAndData();
    _notificationPollingTimer = Timer.periodic(
      const Duration(seconds: 6),
      (_) {
        _loadUnreadNotificationsCount();
        _refreshMatchStatus();
      },
    );
    _searchController.addListener(() {
      final text = _searchController.text;
      if (text != _filterState.searchQuery) {
        setState(() {
          _filterState = _filterState.copyWith(searchQuery: text);
        });
        _searchDebounceTimer?.cancel();
        _searchDebounceTimer = Timer(const Duration(milliseconds: 400), () {
          _filterState.saveToStorage();
          _loadMatchedJobs();
        });
      }
    });
    _scrollController.addListener(_onScroll);
  }

  Future<void> _checkForUpdate() async {
    final update = await _updateCheckerService.checkForUpdate();
    if (!mounted || update == null) return;
    setState(() => _pendingUpdate = update);
  }

  Future<void> _initializeFilterAndData() async {
    final savedFilters = await JobFilterState.loadFromStorage();
    if (!mounted) return;

    setState(() {
      _filterState = savedFilters;
      _searchController.text = savedFilters.searchQuery;
    });

    await _loadMatchedJobs();
    await _loadUnreadNotificationsCount();
    _checkForUpdate();
  }

  Future<void> _refreshMatchStatus() async {
    final status = await _apiService.fetchMatchStatus();
    if (!mounted) return;
    final isEvaluating = status['is_evaluating'] == true;
    final pendingCount = (status['pending_count'] as num?)?.toInt() ?? 0;

    if (_isMatchEngineRunning != isEvaluating || _pendingMatchCount != pendingCount) {
      final wasEvaluating = _isMatchEngineRunning;
      setState(() {
        _isMatchEngineRunning = isEvaluating;
        _pendingMatchCount = pendingCount;
      });

      if (wasEvaluating && !isEvaluating) {
        _loadMatchedJobs();
      }
    }
  }

  Future<void> _loadUnreadNotificationsCount() async {
    final count = await _apiService.fetchUnreadNotificationsCount();

    if (!_hasInitialNotificationSync) {
      _hasInitialNotificationSync = true;
      final initialNotifications = await _apiService.fetchNotifications();
      for (final item in initialNotifications) {
        final id = item['id']?.toString() ?? '';
        if (id.isNotEmpty) _seenNotificationIds.add(id);
      }
    } else if (count > _unreadNotificationCount) {
      final notifications = await _apiService.fetchNotifications();
      for (final item in notifications) {
        final id = item['id']?.toString() ?? '';
        final isRead = item['is_read'] == true;
        if (!isRead && !_seenNotificationIds.contains(id) && id.isNotEmpty) {
          _seenNotificationIds.add(id);
          final title = item['title']?.toString() ?? 'Job Cruiser';
          final message = item['message']?.toString() ?? '';
          NotificationService.instance.showLocalNotification(
            id: id.hashCode,
            title: title,
            body: message,
          );
        }
      }
    }

    if (!mounted) return;
    setState(() => _unreadNotificationCount = count);
  }

  void _onScroll() {
    if (_scrollController.position.pixels >=
            _scrollController.position.maxScrollExtent - 200 &&
        !_isFetchingMore &&
        !_isLoading &&
        _hasMore) {
      _loadMoreJobs();
    }
  }

  @override
  void didUpdateWidget(covariant MyHomePage oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.refreshTrigger != widget.refreshTrigger) {
      oldWidget.refreshTrigger?.removeListener(_scrollToTopAndRefresh);
      widget.refreshTrigger?.addListener(_scrollToTopAndRefresh);
    }
  }

  @override
  void dispose() {
    widget.refreshTrigger?.removeListener(_scrollToTopAndRefresh);
    _notificationPollingTimer?.cancel();
    _searchDebounceTimer?.cancel();
    _searchController.dispose();
    _scrollController.dispose();
    super.dispose();
  }

  void _scrollToTopAndRefresh() {
    if (_scrollController.hasClients) {
      _scrollController.animateTo(
        0.0,
        duration: const Duration(milliseconds: 300),
        curve: Curves.easeOut,
      );
    }
    _loadMatchedJobs();
  }

  Future<void> _loadMatchedJobs() async {
    setState(() {
      _isLoading = true;
      _hasMore = true;
    });

    _loadUnreadNotificationsCount();

    final statusFuture = _apiService.fetchMatchStatus();
    final jobsFuture = _apiService.fetchMatchedJobs(
      minScore: _filterState.minScore,
      maxScore: _filterState.maxScore,
      days: _filterState.recencyDays,
      matchScope: _filterState.matchScope,
      remoteOnly: _filterState.workModel == 'remote_only',
      viewedOnly: _filterState.viewMode == 'viewed',
      unviewedOnly: _filterState.viewMode == 'unviewed',
      sortBy: _filterState.sortBy,
      applicationStatus: _filterState.applicationStatus,
      searchQuery: _filterState.searchQuery,
      offset: 0,
      limit: 50,
    );

    final results = await Future.wait([statusFuture, jobsFuture]);
    final status = results[0] as Map<String, dynamic>;
    final jobs = results[1] as List<MatchedJob>;

    if (!mounted) return;

    setState(() {
      _matchedJobs = jobs;
      _isMatchEngineRunning = status['is_evaluating'] == true;
      _pendingMatchCount = (status['pending_count'] as num?)?.toInt() ?? 0;
      _isLoading = false;

      if (_matchedJobs.isNotEmpty && _selectedJob == null) {
        _selectedJob = _matchedJobs.first;
      } else if (_matchedJobs.isNotEmpty && _selectedJob != null) {
        final existingIndex = _matchedJobs.indexWhere((j) => j.jobId == _selectedJob!.jobId);
        if (existingIndex != -1) {
          _selectedJob = _matchedJobs[existingIndex];
        } else {
          _selectedJob = _matchedJobs.first;
        }
      }
    });
  }

  Future<void> _loadMoreJobs() async {
    if (!_hasMore || _isFetchingMore) return;

    setState(() {
      _isFetchingMore = true;
    });

    final nextOffset = _matchedJobs.length;
    final moreJobs = await _apiService.fetchMatchedJobs(
      minScore: _filterState.minScore,
      maxScore: _filterState.maxScore,
      days: _filterState.recencyDays,
      matchScope: _filterState.matchScope,
      remoteOnly: _filterState.workModel == 'remote_only',
      viewedOnly: _filterState.viewMode == 'viewed',
      unviewedOnly: _filterState.viewMode == 'unviewed',
      sortBy: _filterState.sortBy,
      applicationStatus: _filterState.applicationStatus,
      searchQuery: _filterState.searchQuery,
      offset: nextOffset,
      limit: 50,
    );

    if (!mounted) return;

    setState(() {
      _isFetchingMore = false;
      if (moreJobs.isEmpty) {
        _hasMore = false;
      } else {
        _matchedJobs.addAll(moreJobs);
      }
    });
  }

  void _onFilterStateUpdated(JobFilterState updated) {
    _searchDebounceTimer?.cancel();
    setState(() {
      _filterState = updated;
      _searchController.text = updated.searchQuery;
    });
    _loadMatchedJobs().then((_) => updated.saveToStorage());
  }

  void _openFilterSettings() {
    JobFilterDialog.show(
      context,
      currentState: _filterState,
      onApply: _onFilterStateUpdated,
    );
  }

  Future<void> _dismissJobFromFeed(MatchedJob job) async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(16)),
        title: const Row(
          children: [
            Icon(Icons.visibility_off_outlined, color: AppColors.error),
            SizedBox(width: 8),
            Text('Hide this Job?'),
          ],
        ),
        content: Text(
          'Hide "${job.title}" at "${job.company}" from your feed? It will stay safely stored in the database, but won\'t appear in your matches.',
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(ctx).pop(false),
            child: const Text('Cancel'),
          ),
          ElevatedButton(
            onPressed: () => Navigator.of(ctx).pop(true),
            style: ElevatedButton.styleFrom(
              backgroundColor: AppColors.error,
              foregroundColor: Colors.white,
              shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(8)),
            ),
            child: const Text('Hide Job'),
          ),
        ],
      ),
    );

    if (confirmed != true || !mounted) return;

    setState(() {
      _matchedJobs.removeWhere((j) => j.jobId == job.jobId);
      if (_selectedJob?.jobId == job.jobId) {
        _selectedJob = _matchedJobs.isNotEmpty ? _matchedJobs.first : null;
      }
    });

    final success = await _apiService.dismissJob(job.jobId);
    if (!mounted) return;

    if (success) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text('Job "${job.title}" hidden from feed'),
          action: SnackBarAction(
            label: 'Undo',
            textColor: Colors.amber,
            onPressed: () async {
              await _apiService.undismissJob(job.jobId);
              _loadMatchedJobs();
            },
          ),
        ),
      );
    }
  }

  List<MatchedJob> get _filteredJobs => _matchedJobs;

  void _markJobAsViewedLocally(MatchedJob targetJob) {
    if (targetJob.jobId.isEmpty) return;

    _apiService.markJobAsViewed(targetJob.jobId);

    setState(() {
      final index = _matchedJobs.indexWhere((j) => j.jobId == targetJob.jobId);
      if (index != -1) {
        _matchedJobs[index] = _matchedJobs[index].copyWith(isViewed: true, isNew: false);
      }
      if (_selectedJob?.jobId == targetJob.jobId) {
        _selectedJob = _selectedJob?.copyWith(isViewed: true, isNew: false);
      }
    });
  }

  @override
  Widget build(BuildContext context) {
    final showUpdateBanner =
        _pendingUpdate != null && !_updateBannerDismissed;

    return LayoutBuilder(
      builder: (context, constraints) {
        final isDesktop = constraints.maxWidth >= 960;
        return Scaffold(
          appBar: _buildAppBar(isDesktop),
          body: Column(
            children: [
              if (showUpdateBanner)
                UpdateBanner(
                  update: _pendingUpdate!,
                  onDismiss: () => setState(() => _updateBannerDismissed = true),
                ),
              Expanded(
                child: isDesktop
                    ? _buildDesktopSplitLayout()
                    : _buildMobileFeedLayout(),
              ),
            ],
          ),
        );
      },
    );
  }

  PreferredSizeWidget _buildAppBar(bool isDesktop) {
    return AppBar(
      backgroundColor: AppColors.surface,
      elevation: 0,
      scrolledUnderElevation: 0,
      bottom: PreferredSize(
        preferredSize: const Size.fromHeight(1.0),
        child: Container(
          color: AppColors.outlineVariant.withValues(alpha: 0.5),
          height: 1.0,
        ),
      ),
      title: Row(
        children: [
          Container(
            width: 32,
            height: 32,
            decoration: const BoxDecoration(
              shape: BoxShape.circle,
              color: AppColors.surfaceContainerHigh,
            ),
            child: const Icon(
              Icons.auto_awesome,
              size: 18,
              color: AppColors.primary,
            ),
          ),
          const SizedBox(width: 12),
          const Text(
            'Matched Jobs Inbox',
            style: TextStyle(
              fontSize: 19,
              fontWeight: FontWeight.bold,
              color: AppColors.primary,
            ),
          ),
        ],
      ),
      actions: [
        Stack(
          alignment: Alignment.center,
          children: [
            IconButton(
              icon: const Icon(Icons.notifications_outlined, color: AppColors.primary),
              tooltip: 'Notifications',
              onPressed: () async {
                await showNotificationsSheet(context);
                _loadUnreadNotificationsCount();
              },
            ),
            if (_unreadNotificationCount > 0)
              Positioned(
                top: 8,
                right: 8,
                child: Container(
                  padding: const EdgeInsets.all(4),
                  decoration: const BoxDecoration(
                    color: Colors.redAccent,
                    shape: BoxShape.circle,
                  ),
                  constraints: const BoxConstraints(
                    minWidth: 16,
                    minHeight: 16,
                  ),
                  child: Text(
                    '$_unreadNotificationCount',
                    style: const TextStyle(
                      color: Colors.white,
                      fontSize: 10,
                      fontWeight: FontWeight.bold,
                    ),
                    textAlign: TextAlign.center,
                  ),
                ),
              ),
          ],
        ),
        const SizedBox(width: 8),
      ],
    );
  }

  Widget _buildDesktopSplitLayout() {
    final jobs = _filteredJobs;

    return Column(
      children: [
        if (_isMatchEngineRunning) _buildBackgroundStatusBanner(),
        Expanded(
          child: Row(
            children: [
              SizedBox(
                width: 440,
                child: Container(
                  decoration: BoxDecoration(
                    color: AppColors.surface,
                    border: Border(
                      right: BorderSide(
                        color: AppColors.outlineVariant.withValues(alpha: 0.5),
                        width: 1,
                      ),
                    ),
                  ),
                  child: Column(
                    children: [
                      _buildSearchBar(),
                      JobFilterBar(
                        filterState: _filterState,
                        onFilterChanged: _onFilterStateUpdated,
                        onOpenFilterDialog: _openFilterSettings,
                      ),
                      const Divider(height: 1, color: AppColors.outlineVariant),
                      Expanded(
                        child: _isLoading
                            ? const Center(child: CircularProgressIndicator())
                            : jobs.isEmpty
                                ? _buildEmptyState()
                                : ListView.builder(
                                    controller: _scrollController,
                                    itemCount: jobs.length + (_isFetchingMore ? 1 : 0),
                                    itemBuilder: (context, index) {
                                      if (index == jobs.length) {
                                        return const Padding(
                                          padding: EdgeInsets.all(16.0),
                                          child: Center(
                                            child: CircularProgressIndicator(strokeWidth: 2),
                                          ),
                                        );
                                      }
                                      final job = jobs[index];
                                      final isSelected = _selectedJob?.jobId == job.jobId;
                                      return _buildMatchItem(job, isSelected: isSelected);
                                    },
                                  ),
                      ),
                    ],
                  ),
                ),
              ),
              Expanded(
                child: _selectedJob == null
                    ? _buildNoJobSelectedPlaceholder()
                    : JobDetailPanel(
                        key: ValueKey(_selectedJob!.jobId),
                        job: _selectedJob!,
                        showBackButton: false,
                        onStatusChanged: (newStatus) {
                          setState(() {
                            final idx = _matchedJobs.indexWhere((j) => j.jobId == _selectedJob!.jobId);
                            if (idx != -1) {
                              _matchedJobs[idx] = _matchedJobs[idx].copyWith(applicationStatus: newStatus);
                            }
                            _selectedJob = _selectedJob?.copyWith(applicationStatus: newStatus);
                          });
                        },
                        onJobDismissed: (dismissedJob) {
                          setState(() {
                            _matchedJobs.removeWhere((j) => j.jobId == dismissedJob.jobId);
                            _selectedJob = _matchedJobs.isNotEmpty ? _matchedJobs.first : null;
                          });
                        },
                      ),
              ),
            ],
          ),
        ),
      ],
    );
  }

  Widget _buildMobileFeedLayout() {
    final jobs = _filteredJobs;

    return RefreshIndicator(
      onRefresh: _loadMatchedJobs,
      color: AppColors.primary,
      child: Column(
        children: [
          if (_isMatchEngineRunning) _buildBackgroundStatusBanner(),
          _buildSearchBar(),
          JobFilterBar(
            filterState: _filterState,
            onFilterChanged: _onFilterStateUpdated,
            onOpenFilterDialog: _openFilterSettings,
          ),
          const Divider(height: 1, color: AppColors.outlineVariant),
          Expanded(
            child: _isLoading
                ? const Center(child: CircularProgressIndicator())
                : jobs.isEmpty
                    ? _buildEmptyState()
                    : ListView.builder(
                        controller: _scrollController,
                        itemCount: jobs.length + (_isFetchingMore ? 1 : 0),
                        itemBuilder: (context, index) {
                          if (index == jobs.length) {
                            return const Padding(
                              padding: EdgeInsets.all(16.0),
                              child: Center(
                                child: CircularProgressIndicator(strokeWidth: 2),
                              ),
                            );
                          }
                          final job = jobs[index];
                          return _buildMatchItem(job);
                        },
                      ),
          ),
        ],
      ),
    );
  }

  Widget _buildNoJobSelectedPlaceholder() {
    return Center(
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          Container(
            width: 72,
            height: 72,
            decoration: BoxDecoration(
              shape: BoxShape.circle,
              color: AppColors.primary.withValues(alpha: 0.05),
            ),
            child: const Icon(
              Icons.touch_app_outlined,
              size: 36,
              color: AppColors.outline,
            ),
          ),
          const SizedBox(height: 16),
          const Text(
            'Select a Job to View Deep Dive',
            style: TextStyle(
              fontSize: 18,
              fontWeight: FontWeight.w700,
              color: AppColors.primary,
            ),
          ),
          const SizedBox(height: 6),
          const Text(
            'Select any listing from the left feed to view full requirements, AI reasoning, and tailor documents.',
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

  Widget _buildBackgroundStatusBanner() {
    final message = _pendingMatchCount > 0
        ? 'AI Match Engine is processing $_pendingMatchCount pending jobs against your profile. Pull down to refresh.'
        : 'AI Match Engine is processing jobs against your profile. Pull down to refresh.';

    return Container(
      width: double.infinity,
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
      color: AppColors.primaryContainer.withValues(alpha: 0.15),
      child: Row(
        children: [
          const SizedBox(
            width: 14,
            height: 14,
            child: CircularProgressIndicator(strokeWidth: 2),
          ),
          const SizedBox(width: 10),
          Expanded(
            child: Text(
              message,
              style: const TextStyle(
                fontSize: 12,
                color: AppColors.primary,
                fontWeight: FontWeight.w500,
              ),
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildSearchBar() {
    return Container(
      color: AppColors.surface,
      padding: const EdgeInsets.symmetric(horizontal: 16.0, vertical: 8.0),
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 12.0, vertical: 2.0),
        decoration: BoxDecoration(
          color: AppColors.surfaceContainerLowest,
          borderRadius: BorderRadius.circular(8.0),
          border: Border.all(color: AppColors.outlineVariant),
          boxShadow: [
            BoxShadow(
              color: Colors.black.withValues(alpha: 0.02),
              blurRadius: 2,
              offset: const Offset(0, 1),
            )
          ],
        ),
        child: Row(
          children: [
            const Icon(
              Icons.search,
              color: AppColors.onSurfaceVariant,
              size: 20,
            ),
            const SizedBox(width: 8),
            Expanded(
              child: TextField(
                controller: _searchController,
                decoration: const InputDecoration(
                  hintText: 'Search roles, skills, companies...',
                  hintStyle: TextStyle(
                    color: AppColors.onSurfaceVariant,
                    fontSize: 13,
                    fontWeight: FontWeight.w400,
                  ),
                  border: InputBorder.none,
                  isDense: true,
                ),
                style: const TextStyle(
                  color: AppColors.primary,
                  fontSize: 13,
                ),
              ),
            ),
            if (_searchController.text.isNotEmpty)
              IconButton(
                icon: const Icon(Icons.clear, size: 16),
                onPressed: () {
                  _searchController.clear();
                  _onFilterStateUpdated(_filterState.copyWith(searchQuery: ''));
                },
              ),
          ],
        ),
      ),
    );
  }

  Widget _buildEmptyState() {
    final hasActiveFilters = !_filterState.isDefault;

    return SingleChildScrollView(
      physics: const AlwaysScrollableScrollPhysics(),
      child: Container(
        padding: const EdgeInsets.all(32),
        alignment: Alignment.center,
        child: Column(
          children: [
            const SizedBox(height: 40),
            const Icon(
              Icons.auto_awesome_outlined,
              size: 56,
              color: AppColors.outline,
            ),
            const SizedBox(height: 16),
            Text(
              hasActiveFilters ? 'No Matching Jobs Found' : 'No AI Matched Jobs Yet',
              style: const TextStyle(
                fontSize: 17,
                fontWeight: FontWeight.w700,
                color: AppColors.primary,
              ),
            ),
            const SizedBox(height: 8),
            Text(
              hasActiveFilters
                  ? 'No listings match your active filters or search terms. Try adjusting your match score range, recency, or scope.'
                  : 'Set up your target roles, min salary, and industries to run AI match evaluations.',
              textAlign: TextAlign.center,
              style: const TextStyle(
                fontSize: 13,
                color: AppColors.onSurfaceVariant,
              ),
            ),
            const SizedBox(height: 20),
            if (hasActiveFilters)
              ElevatedButton.icon(
                onPressed: () => _onFilterStateUpdated(_filterState.reset()),
                icon: const Icon(Icons.refresh, size: 16),
                label: const Text('Reset All Filters'),
                style: ElevatedButton.styleFrom(
                  backgroundColor: AppColors.primary,
                  foregroundColor: Colors.white,
                  padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 10),
                ),
              )
            else
              ElevatedButton.icon(
                onPressed: () async {
                  await Navigator.of(context).push(
                    MaterialPageRoute(
                      builder: (_) => const preferences_page.SetPreferencesScreen(),
                    ),
                  );
                  _loadMatchedJobs();
                },
                icon: const Icon(Icons.tune, size: 16),
                label: const Text('Set Match Preferences'),
                style: ElevatedButton.styleFrom(
                  backgroundColor: AppColors.primary,
                  foregroundColor: Colors.white,
                  padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 10),
                ),
              ),
          ],
        ),
      ),
    );
  }

  Widget _buildMatchItem(MatchedJob job, {bool isSelected = false}) {
    final isHighMatch = job.matchScore >= 80;

    return InkWell(
      onTap: () {
        _markJobAsViewedLocally(job);
        final screenWidth = MediaQuery.of(context).size.width;
        if (screenWidth >= 960) {
          setState(() {
            _selectedJob = job;
          });
        } else {
          widget.onSelectJob(job);
        }
      },
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 16.0, vertical: 12.0),
        decoration: BoxDecoration(
          color: isSelected
              ? AppColors.surfaceContainerLowest
              : AppColors.surface,
          border: Border(
            bottom: const BorderSide(color: AppColors.outlineVariant, width: 1.0),
            left: isSelected
                ? const BorderSide(color: AppColors.primary, width: 4.0)
                : BorderSide.none,
          ),
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              crossAxisAlignment: CrossAxisAlignment.center,
              children: [
                Expanded(
                  child: Text(
                    job.title,
                    style: TextStyle(
                      fontSize: 14.5,
                      fontWeight: isSelected ? FontWeight.w800 : FontWeight.w700,
                      color: AppColors.primary,
                      letterSpacing: -0.2,
                    ),
                    maxLines: 2,
                    overflow: TextOverflow.ellipsis,
                  ),
                ),
                const SizedBox(width: 8),
                Row(
                  mainAxisSize: MainAxisSize.min,
                  crossAxisAlignment: CrossAxisAlignment.center,
                  children: [
                    Container(
                      padding: const EdgeInsets.symmetric(horizontal: 7, vertical: 3),
                      decoration: BoxDecoration(
                        color: isHighMatch
                            ? AppColors.matchGreen
                            : AppColors.surfaceContainerLowest,
                        borderRadius: BorderRadius.circular(16),
                        border: isHighMatch
                            ? null
                            : Border.all(color: AppColors.outline, width: 1),
                      ),
                      child: Text(
                        job.matchScore > 0
                            ? '${job.matchScore}%'
                            : 'Unmatched',
                        style: TextStyle(
                          fontSize: 10.5,
                          fontWeight: FontWeight.w700,
                          color: isHighMatch
                              ? AppColors.onTertiary
                              : AppColors.outline,
                        ),
                      ),
                    ),
                    if (job.isNew) ...[
                      const SizedBox(width: 4),
                      Container(
                        padding: const EdgeInsets.symmetric(horizontal: 5, vertical: 2),
                        decoration: BoxDecoration(
                          color: AppColors.primaryContainer,
                          borderRadius: BorderRadius.circular(4),
                        ),
                        child: const Text(
                          'NEW',
                          style: TextStyle(
                            fontSize: 8.5,
                            fontWeight: FontWeight.w800,
                            color: Colors.white,
                            letterSpacing: 0.5,
                          ),
                        ),
                      ),
                    ],
                    const SizedBox(width: 2),
                    SizedBox(
                      width: 26,
                      height: 26,
                      child: PopupMenuButton<String>(
                        icon: const Icon(Icons.more_vert, size: 16, color: AppColors.outline),
                        padding: EdgeInsets.zero,
                        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
                        onSelected: (action) {
                          if (action == 'details') {
                            widget.onSelectJob(job);
                          } else if (action == 'dismiss') {
                            _dismissJobFromFeed(job);
                          }
                        },
                        itemBuilder: (context) => [
                          const PopupMenuItem(
                            value: 'details',
                            child: Row(
                              children: [
                                Icon(Icons.open_in_new, size: 15, color: AppColors.primary),
                                SizedBox(width: 8),
                                Text('Open in Separate View'),
                              ],
                            ),
                          ),
                          const PopupMenuDivider(),
                          const PopupMenuItem(
                            value: 'dismiss',
                            child: Row(
                              children: [
                                Icon(Icons.visibility_off_outlined, size: 15, color: AppColors.error),
                                SizedBox(width: 8),
                                Text(
                                  'Hide Job',
                                  style: TextStyle(color: AppColors.error),
                                ),
                              ],
                            ),
                          ),
                        ],
                      ),
                    ),
                  ],
                ),
              ],
            ),
            const SizedBox(height: 6),
            Row(
              children: [
                CompanyLogoAvatar(
                  companyName: job.company,
                  jobUrl: job.url,
                  size: 18,
                ),
                const SizedBox(width: 6),
                Flexible(
                  flex: 2,
                  child: Text(
                    job.company,
                    style: const TextStyle(
                      fontSize: 12.5,
                      fontWeight: FontWeight.w600,
                      color: AppColors.onSurfaceVariant,
                    ),
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                  ),
                ),
                if (job.location.isNotEmpty) ...[
                  const SizedBox(width: 6),
                  Icon(
                    job.isRemote ? Icons.wifi : Icons.location_on_outlined,
                    size: 11,
                    color: AppColors.onSurfaceVariant,
                  ),
                  const SizedBox(width: 2),
                  Flexible(
                    flex: 1,
                    child: Text(
                      job.location,
                      style: const TextStyle(
                        fontSize: 11,
                        fontWeight: FontWeight.w500,
                        color: AppColors.onSurfaceVariant,
                      ),
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                    ),
                  ),
                ],
                if (job.scrapedAgoText.isNotEmpty) ...[
                  const SizedBox(width: 6),
                  Text(
                    '• ${job.scrapedAgoText}',
                    style: const TextStyle(
                      fontSize: 10.5,
                      fontWeight: FontWeight.w500,
                      color: AppColors.onSurfaceVariant,
                    ),
                  ),
                ],
              ],
            ),
            if (job.matchReasoning.isNotEmpty || job.summary.isNotEmpty) ...[
              const SizedBox(height: 6),
              Text(
                job.matchReasoning.isNotEmpty
                    ? job.matchReasoning
                    : job.summary,
                style: const TextStyle(
                  fontSize: 12,
                  height: 1.35,
                  fontWeight: FontWeight.w400,
                  color: AppColors.secondary,
                ),
                maxLines: 2,
                overflow: TextOverflow.ellipsis,
              ),
            ],
          ],
        ),
      ),
    );
  }
}
