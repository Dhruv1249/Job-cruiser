import 'package:flutter/material.dart';
import 'details.dart' as details_page;
import 'profile.dart';
import 'preferences.dart' as preferences_page;
import 'auth.dart';
import 'onboarding.dart';
import 'tracker.dart';
import 'models/job.dart';
import 'services/api_service.dart';
import 'widgets/company_logo_avatar.dart';
import 'package:flutter_dotenv/flutter_dotenv.dart';

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();
  await dotenv.load(fileName: ".env");
  runApp(const MyApp());
}

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

class JobCruiserShell extends StatefulWidget {
  const JobCruiserShell({super.key});

  @override
  State<JobCruiserShell> createState() => _JobCruiserShellState();
}

class _JobCruiserShellState extends State<JobCruiserShell> {
  int _currentIndex = 0;

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
    return Scaffold(
      body: IndexedStack(
        index: _currentIndex,
        children: [
          MyHomePage(onSelectJob: _openJobDetails),
          const ApplicationTrackerPage(),
          const ProfilePage(),
        ],
      ),
      bottomNavigationBar: _buildBottomNav(),
    );
  }

  Widget _buildBottomNav() {
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
          setState(() {
            _currentIndex = index;
          });
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

class MyHomePage extends StatefulWidget {
  const MyHomePage({super.key, required this.onSelectJob});

  final Function(MatchedJob job) onSelectJob;

  @override
  State<MyHomePage> createState() => _MyHomePageState();
}

class _MyHomePageState extends State<MyHomePage> {
  final ApiService _apiService = ApiService();
  final TextEditingController _searchController = TextEditingController();
  final ScrollController _scrollController = ScrollController();

  List<MatchedJob> _matchedJobs = [];
  bool _isLoading = true;
  bool _isFetchingMore = false;
  bool _hasMore = true;
  int _minScoreFilter = 0;
  String _viewFilterMode = 'all'; // 'all', 'unviewed', 'viewed'
  String _searchQuery = '';

  @override
  void initState() {
    super.initState();
    _loadMatchedJobs();
    _searchController.addListener(() {
      setState(() {
        _searchQuery = _searchController.text.toLowerCase();
      });
    });
    _scrollController.addListener(_onScroll);
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
  void dispose() {
    _searchController.dispose();
    _scrollController.dispose();
    super.dispose();
  }

  Future<void> _loadMatchedJobs() async {
    setState(() {
      _isLoading = true;
      _hasMore = true;
    });

    final jobs = await _apiService.fetchMatchedJobs(
      minScore: _minScoreFilter,
      viewedOnly: _viewFilterMode == 'viewed',
      unviewedOnly: _viewFilterMode == 'unviewed',
      offset: 0,
      limit: 50,
    );

    if (!mounted) return;

    setState(() {
      _matchedJobs = jobs;
      _isLoading = false;
    });
  }

  Future<void> _loadMoreJobs() async {
    if (!_hasMore || _isFetchingMore) return;

    setState(() {
      _isFetchingMore = true;
    });

    final nextOffset = _matchedJobs.length;
    final moreJobs = await _apiService.fetchMatchedJobs(
      minScore: _minScoreFilter,
      viewedOnly: _viewFilterMode == 'viewed',
      unviewedOnly: _viewFilterMode == 'unviewed',
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

  List<MatchedJob> get _filteredJobs {
    List<MatchedJob> list = List.from(_matchedJobs);

    if (_viewFilterMode == 'unviewed') {
      list = list.where((job) => !job.isViewed).toList();
    } else if (_viewFilterMode == 'viewed') {
      list = list.where((job) => job.isViewed).toList();
    }

    if (_minScoreFilter > 0) {
      list = list.where((job) => job.matchScore >= _minScoreFilter).toList();
    }

    if (_searchQuery.isNotEmpty) {
      list = list.where((job) {
        final titleMatch = job.title.toLowerCase().contains(_searchQuery);
        final companyMatch = job.company.toLowerCase().contains(_searchQuery);
        final summaryMatch = job.summary.toLowerCase().contains(_searchQuery);
        final techMatch = job.techStack.any(
          (tech) => tech.toLowerCase().contains(_searchQuery),
        );
        return titleMatch || companyMatch || summaryMatch || techMatch;
      }).toList();
    }

    return list;
  }

  void _markJobAsViewedLocally(MatchedJob targetJob) {
    if (targetJob.jobId.isEmpty) return;

    _apiService.markJobAsViewed(targetJob.jobId);

    setState(() {
      final index = _matchedJobs.indexWhere((j) => j.jobId == targetJob.jobId);
      if (index != -1) {
        _matchedJobs[index] = _matchedJobs[index].copyWith(isViewed: true, isNew: false);
      }
    });
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: _buildAppBar(),
      body: RefreshIndicator(
        onRefresh: _loadMatchedJobs,
        color: AppColors.primary,
        child: Column(
          children: [
            _buildBackgroundStatusBanner(),
            _buildSearchBar(),
            _buildScoreFilterChips(),
            Expanded(
              child: _isLoading
                  ? const Center(child: CircularProgressIndicator())
                  : _filteredJobs.isEmpty
                      ? _buildEmptyState()
                      : ListView.builder(
                          controller: _scrollController,
                          itemCount: _filteredJobs.length + (_isFetchingMore ? 1 : 0),
                          itemBuilder: (context, index) {
                            if (index == _filteredJobs.length) {
                              return const Padding(
                                padding: EdgeInsets.all(16.0),
                                child: Center(
                                  child: CircularProgressIndicator(
                                    strokeWidth: 2,
                                  ),
                                ),
                              );
                            }
                            final job = _filteredJobs[index];
                            return _buildMatchItem(job);
                          },
                        ),
            ),
          ],
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
        preferredSize: const Size.fromHeight(1.0),
        child: Container(
          color: AppColors.outlineVariant,
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
              fontSize: 20,
              fontWeight: FontWeight.bold,
              color: AppColors.primary,
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildBackgroundStatusBanner() {
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
      color: AppColors.primaryContainer.withValues(alpha: 0.15),
      child: const Row(
        children: [
          SizedBox(
            width: 14,
            height: 14,
            child: CircularProgressIndicator(strokeWidth: 2),
          ),
          SizedBox(width: 10),
          Expanded(
            child: Text(
              'AI Match Engine is processing 10,000+ jobs against your profile. Pull down to refresh.',
              style: TextStyle(
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
      padding: const EdgeInsets.symmetric(horizontal: 20.0, vertical: 8.0),
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
                  hintText: 'Search matches, skills, companies...',
                  hintStyle: TextStyle(
                    color: AppColors.onSurfaceVariant,
                    fontSize: 14,
                    fontWeight: FontWeight.w400,
                  ),
                  border: InputBorder.none,
                  isDense: true,
                ),
                style: const TextStyle(
                  color: AppColors.primary,
                  fontSize: 14,
                ),
              ),
            ),
            if (_searchController.text.isNotEmpty)
              IconButton(
                icon: const Icon(Icons.clear, size: 18),
                onPressed: () {
                  _searchController.clear();
                },
              ),
          ],
        ),
      ),
    );
  }

  Widget _buildScoreFilterChips() {
    return Container(
      color: AppColors.surface,
      height: 44,
      child: SingleChildScrollView(
        scrollDirection: Axis.horizontal,
        padding: const EdgeInsets.symmetric(horizontal: 20),
        child: Row(
          children: [
            const Text(
              'View: ',
              style: TextStyle(
                fontSize: 12,
                fontWeight: FontWeight.w600,
                color: AppColors.onSurfaceVariant,
              ),
            ),
            _buildViewFilterChip('All Matches', 'all'),
            const SizedBox(width: 6),
            _buildViewFilterChip('Unviewed', 'unviewed'),
            const SizedBox(width: 6),
            _buildViewFilterChip('Viewed', 'viewed'),
            const SizedBox(width: 12),
            Container(height: 16, width: 1, color: AppColors.outlineVariant),
            const SizedBox(width: 12),
            _buildScoreFilterChip('80%+ Top', 80),
            const SizedBox(width: 6),
            _buildScoreFilterChip('60%+ Good', 60),
          ],
        ),
      ),
    );
  }

  Widget _buildViewFilterChip(String label, String mode) {
    final isSelected = _viewFilterMode == mode;
    return ChoiceChip(
      label: Text(
        label,
        style: TextStyle(
          fontSize: 11,
          fontWeight: FontWeight.w600,
          color: isSelected
              ? AppColors.surfaceContainerLowest
              : AppColors.onSurfaceVariant,
        ),
      ),
      selected: isSelected,
      selectedColor: AppColors.primary,
      backgroundColor: AppColors.surfaceContainerLowest,
      side: BorderSide(
        color: isSelected ? AppColors.primary : AppColors.outlineVariant,
      ),
      onSelected: (selected) {
        if (selected) {
          setState(() {
            _viewFilterMode = mode;
          });
          _loadMatchedJobs();
        }
      },
    );
  }

  Widget _buildScoreFilterChip(String label, int minScore) {
    final isSelected = _minScoreFilter == minScore;
    return ChoiceChip(
      label: Text(
        label,
        style: TextStyle(
          fontSize: 11,
          fontWeight: FontWeight.w600,
          color: isSelected
              ? AppColors.surfaceContainerLowest
              : AppColors.onSurfaceVariant,
        ),
      ),
      selected: isSelected,
      selectedColor: AppColors.primary,
      backgroundColor: AppColors.surfaceContainerLowest,
      side: BorderSide(
        color: isSelected ? AppColors.primary : AppColors.outlineVariant,
      ),
      onSelected: (selected) {
        if (selected) {
          setState(() {
            _minScoreFilter = isSelected ? 0 : minScore;
          });
          _loadMatchedJobs();
        }
      },
    );
  }

  Widget _buildEmptyState() {
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
              size: 64,
              color: AppColors.outline,
            ),
            const SizedBox(height: 16),
            const Text(
              'No AI Matched Jobs Yet',
              style: TextStyle(
                fontSize: 18,
                fontWeight: FontWeight.w700,
                color: AppColors.primary,
              ),
            ),
            const SizedBox(height: 8),
            const Text(
              'Set up your target roles, min salary, and industries to run AI match evaluations.',
              textAlign: TextAlign.center,
              style: TextStyle(
                fontSize: 14,
                color: AppColors.onSurfaceVariant,
              ),
            ),
            const SizedBox(height: 24),
            ElevatedButton.icon(
              onPressed: () async {
                await Navigator.of(context).push(
                  MaterialPageRoute(
                    builder: (_) => const preferences_page.SetPreferencesScreen(),
                  ),
                );
                _loadMatchedJobs();
              },
              icon: const Icon(Icons.tune, size: 18),
              label: const Text('Set Match Preferences'),
              style: ElevatedButton.styleFrom(
                backgroundColor: AppColors.primary,
                foregroundColor: Colors.white,
                padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 12),
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildMatchItem(MatchedJob job) {
    final isHighMatch = job.matchScore >= 80;

    return InkWell(
      onTap: () {
        _markJobAsViewedLocally(job);
        widget.onSelectJob(job);
      },
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 16.0, vertical: 14.0),
        decoration: const BoxDecoration(
          color: AppColors.surface,
          border: Border(
            bottom: BorderSide(color: AppColors.outlineVariant, width: 1.0),
          ),
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              mainAxisAlignment: MainAxisAlignment.spaceBetween,
              children: [
                Expanded(
                  child: Text(
                    job.title,
                    style: const TextStyle(
                      fontSize: 15,
                      fontWeight: FontWeight.w700,
                      color: AppColors.primary,
                      letterSpacing: -0.2,
                    ),
                    maxLines: 2,
                    overflow: TextOverflow.ellipsis,
                  ),
                ),
                const SizedBox(width: 10),
                Column(
                  crossAxisAlignment: CrossAxisAlignment.end,
                  children: [
                    Container(
                      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
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
                            ? '${job.matchScore}% Match'
                            : 'Unmatched',
                        style: TextStyle(
                          fontSize: 11,
                          fontWeight: FontWeight.w600,
                          color: isHighMatch
                              ? AppColors.onTertiary
                              : AppColors.outline,
                        ),
                      ),
                    ),
                    if (job.isNew) ...[
                      const SizedBox(height: 4),
                      Container(
                        padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
                        decoration: BoxDecoration(
                          color: AppColors.primaryContainer,
                          borderRadius: BorderRadius.circular(4),
                        ),
                        child: const Text(
                          'NEW',
                          style: TextStyle(
                            fontSize: 9,
                            fontWeight: FontWeight.w800,
                            color: Colors.white,
                            letterSpacing: 0.5,
                          ),
                        ),
                      ),
                    ],
                  ],
                ),
              ],
            ),
            const SizedBox(height: 8),
            Row(
              children: [
                CompanyLogoAvatar(
                  companyName: job.company,
                  jobUrl: job.url,
                  size: 20,
                ),
                const SizedBox(width: 6),
                Flexible(
                  flex: 2,
                  child: Text(
                    job.company,
                    style: const TextStyle(
                      fontSize: 13,
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
                    size: 12,
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
                if (job.postedDate.isNotEmpty) ...[
                  const SizedBox(width: 6),
                  Text(
                    '• ${job.postedDate}',
                    style: const TextStyle(
                      fontSize: 11,
                      fontWeight: FontWeight.w500,
                      color: AppColors.onSurfaceVariant,
                    ),
                  ),
                ],
              ],
            ),
            if (job.matchReasoning.isNotEmpty || job.summary.isNotEmpty) ...[
              const SizedBox(height: 8),
              Text(
                job.matchReasoning.isNotEmpty
                    ? job.matchReasoning
                    : job.summary,
                style: const TextStyle(
                  fontSize: 12.5,
                  height: 1.4,
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
