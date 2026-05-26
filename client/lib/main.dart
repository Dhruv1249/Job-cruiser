import 'package:flutter/material.dart';
import 'details.dart' as details_page;
import 'profile.dart';

void main() {
  runApp(const MyApp());
}

class AppColors{
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
  // Custom colors used in the specific styling
  static const Color successGreen = Color(0xFF10B981);
  static const Color slate900 = Color(0xFF0F172A);
  static const Color sliderInactive = Color(0xFFE2E8F0);
}

class MyApp extends StatelessWidget {
  const MyApp({super.key});

  // This widget is the root of your application.
  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      debugShowCheckedModeBanner: false,
      title: 'Inbox',
      theme: ThemeData(
        // This is the theme of your application.
        //
        // TRY THIS: Try running your application with "flutter run". You'll see
        // the application has a purple toolbar. Then, without quitting the app,
        // try changing the seedColor in the colorScheme below to Colors.green
        // and then invoke "hot reload" (save your changes or press the "hot
        // reload" button in a Flutter-supported IDE, or press "r" if you used
        // the command line to start the app).
        //
        // Notice that the counter didn't reset back to zero; the application
        // state is not lost during the reload. To reset the state, use hot
        // restart instead.
        //
        // This works for code too, not just values: Most code changes can be
        // tested with just a hot reload.
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
      home: const JobCruiserShell(),
    );
  }
}

class JobCruiserShell extends StatefulWidget {
  const JobCruiserShell({super.key});

  @override
  State<JobCruiserShell> createState() => _JobCruiserShellState();
}

class _JobCruiserShellState extends State<JobCruiserShell> {
  int _currentIndex = 0;

  void _showInbox() {
    setState(() {
      _currentIndex = 0;
    });
  }

  void _showDetails() {
    setState(() {
      _currentIndex = 1;
    });
  }

  @override
  Widget build(BuildContext context) {
    return PopScope(
      canPop: _currentIndex == 0,
      onPopInvokedWithResult: (didPop, _) {
        if (didPop || _currentIndex == 0) {
          return;
        }

        setState(() {
          _currentIndex = 0;
        });
      },
      child: Scaffold(
        body: IndexedStack(
          index: _currentIndex,
          children: [
            MyHomePage(onOpenDetails: _showDetails),
            details_page.CompanyDetailsPage(onBackToInbox: _showInbox),
            const ProfilePage(),
          ],
        ),
        bottomNavigationBar: _buildBottomNav(),
      ),
    );
  }

  Widget _buildBottomNav() {
    final int navigationIndex = _currentIndex == 2 ? 1 : 0;

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
        selectedIndex: navigationIndex,
        onDestinationSelected: (int index) {
          setState(() {
            _currentIndex = index == 0 ? 0 : 2;
          });
        },
        destinations: const [
          NavigationDestination(
            icon: Icon(Icons.chat_bubble_outline),
            selectedIcon: Icon(Icons.chat_bubble),
            label: 'Inbox',
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

class MyHomePage extends StatelessWidget {
  const MyHomePage({super.key, required this.onOpenDetails});

  final VoidCallback onOpenDetails;

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: _buildAppBar(),
      body: Column(
        children: [
          _buildSearchBar(),
          Expanded(
            child: ListView(
              children: [
                _buildMatchItem(
                  companyName: 'Acme Systems',
                  time: '10:42 AM',
                  description: 'Requires 5+ years React, Node.js & TypeScript.',
                  matchPercentage: '92% Match',
                  isHighMatch: true,
                  avatarUrl:
                      'https://lh3.googleusercontent.com/aida-public/AB6AXuBxCO3VOJZVxhlHbxaj8NfNZbvC7VSSafoDr6gM6GpBIItvw5kqPCXl4raPmDWticXJQVZFW4KY2yidNXi8hPCoqqvWXvmLPbGZVsfef2yiJbut7LY258kZ0hYJXRn6clA6TEKlP6sD8HtYzp0q_auNGvxGNiGRe_W7wJOWKKqG_N4Vv3zUbH_ItDAqe8Z-NkKaRtS7bI108gHLHeauRo0o8phbcr99AcNTDUt1ysjxas-bI6BSGM15zM9IUyiynZaKiM36c0cUwiA',
                ),
                _buildMatchItem(
                  companyName: 'Stratos Financial',
                  time: 'Yesterday',
                  description: 'Looking for a Senior Data Analyst with Python.',
                  matchPercentage: '78% Match',
                  isHighMatch: false,
                  avatarLetter: 'S',
                ),
                _buildMatchItem(
                  companyName: 'Nexus Logistics',
                  time: 'Tuesday',
                  description: 'Entry level project manager role. Hybrid.',
                  matchPercentage: '62% Match',
                  isHighMatch: false,
                  avatarUrl:
                      'https://lh3.googleusercontent.com/aida-public/AB6AXuB7NkZkKZtUOlI790QP4bVQH9vTiLlyiMTOSQNePzV3WyynwzRKe8SvhXpwGZM0X4rT33IZRzytU60m_CCy7Xoal7mfpDZX7yGgeKWJMS-05qDMvr0hy41k980H0Rs1qlsVQjL2yC0C7PwU2qloJ56DYiMfsBbS7d7CVyYt3rhvWdhYNeoVYvH7lTh4-lMSACUM_3cqA42n4Q22NMnaV6Wkaz5fnHttEqkcgWeCA2zUriBEf5lOkLXjMtvu2HYOPamIYszVh5jWYSQ',
                ),
              ],
            ),
          ),
        ],
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
            clipBehavior: Clip.antiAlias,
            child: Image.network(
              'https://lh3.googleusercontent.com/aida-public/AB6AXuCKo4JnnrqSxvI8JBEuU4j1p-I3aaUFpR5GoJ_ROakDYcvUOvPDy7AkpLMbiK8t29d-haFrfbKdA1oBXcXQy9WO2-gFb5QkN4xkRpuSRMl6Oe_Pmo4zrGIdQUvOSXlTR1JcNhyc15838sMA4qRCVWYoXpB4qEubdVCbzZ8iwMvnloi_VsXxWUXiByIzIrJI-ramiwdoH5GLFmvgYw4J_m_S0rqH-pM8QkuOg3WD33Ln0YQjCfXC9-r_Q5n62oAJQct00P6Tfl0Lh3M',
              fit: BoxFit.cover,
            ),
          ),
          const SizedBox(width: 12),
          const Text(
            'Inbox',
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
          ],
        ),
      ),
    );
  }

  Widget _buildMatchItem({
    required String companyName,
    required String time,
    required String description,
    required String matchPercentage,
    required bool isHighMatch,
    String? avatarUrl,
    String? avatarLetter,
  }) {
    return InkWell(
      onTap: onOpenDetails,
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 20.0, vertical: 12.0),
        decoration: const BoxDecoration(
          color: AppColors.surface,
          border: Border(
            bottom: BorderSide(color: AppColors.outlineVariant, width: 1.0),
          ),
        ),
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            // Avatar
            Container(
              width: 40,
              height: 40,
              decoration: BoxDecoration(
                shape: BoxShape.circle,
                color: AppColors.surfaceContainer,
                border: Border.all(color: AppColors.outlineVariant),
              ),
              clipBehavior: Clip.antiAlias,
              child: avatarUrl != null
                  ? Image.network(avatarUrl, fit: BoxFit.cover)
                  : Container(
                      color: AppColors.primaryContainer,
                      child: Center(
                        child: Text(
                          avatarLetter ?? '',
                          style: const TextStyle(
                            color: AppColors.onPrimaryContainer,
                            fontSize: 20,
                            fontWeight: FontWeight.w600,
                          ),
                        ),
                      ),
                    ),
            ),
            const SizedBox(width: 12),
            // Middle Content
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Row(
                    mainAxisAlignment: MainAxisAlignment.spaceBetween,
                    crossAxisAlignment: CrossAxisAlignment.baseline,
                    textBaseline: TextBaseline.alphabetic,
                    children: [
                      Expanded(
                        child: Text(
                          companyName,
                          style: const TextStyle(
                            fontSize: 12,
                            fontWeight: FontWeight.w600,
                            color: AppColors.primary,
                          ),
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                        ),
                      ),
                      const SizedBox(width: 8),
                      Text(
                        time,
                        style: const TextStyle(
                          fontSize: 11,
                          fontWeight: FontWeight.w500,
                          color: AppColors.onSurfaceVariant,
                        ),
                      ),
                    ],
                  ),
                  const SizedBox(height: 4),
                  Text(
                    description,
                    style: const TextStyle(
                      fontSize: 14,
                      fontWeight: FontWeight.w400,
                      color: AppColors.onSurfaceVariant,
                    ),
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                  ),
                ],
              ),
            ),
            const SizedBox(width: 8),
            // Match Percentage Badge
            Container(
              margin: const EdgeInsets.only(top: 4),
              padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
              decoration: BoxDecoration(
                color: isHighMatch
                    ? AppColors.onTertiaryContainer
                    : AppColors.surfaceContainerLowest,
                borderRadius: BorderRadius.circular(16),
                border: isHighMatch
                    ? null
                    : Border.all(color: AppColors.outline, width: 1),
              ),
              child: Text(
                matchPercentage,
                style: TextStyle(
                  fontSize: 11,
                  fontWeight: FontWeight.w600,
                  color: isHighMatch ? AppColors.onTertiary : AppColors.outline,
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }

}