import 'package:flutter/material.dart';
import 'main.dart' show AppColors;
import 'models/job.dart';
import 'widgets/job_detail_panel.dart';

void main() {
  runApp(const CompanyDetailsApp());
}

/// Standalone entry application widget for the Company Details screen.
class CompanyDetailsApp extends StatelessWidget {
  const CompanyDetailsApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Company Deep Dive',
      debugShowCheckedModeBanner: false,
      theme: ThemeData(
        scaffoldBackgroundColor: AppColors.background,
        fontFamily: 'Inter',
        colorScheme: ColorScheme.fromSeed(
          seedColor: AppColors.primary,
          surface: AppColors.surface,
          primary: AppColors.primary,
        ),
      ),
      home: const CompanyDetailsPage(),
    );
  }
}

/// Dedicated page rendering full details of a specific job opportunity.
class CompanyDetailsPage extends StatelessWidget {
  const CompanyDetailsPage({
    super.key,
    this.onBackToInbox,
    this.job,
  });

  final VoidCallback? onBackToInbox;
  final MatchedJob? job;

  @override
  Widget build(BuildContext context) {
    final activeJob = job;

    if (activeJob == null) {
      return Scaffold(
        appBar: AppBar(
          backgroundColor: AppColors.surface,
          leading: IconButton(
            icon: const Icon(Icons.arrow_back, color: AppColors.primary),
            onPressed: () {
              if (onBackToInbox != null) {
                onBackToInbox!();
              } else {
                Navigator.maybePop(context);
              }
            },
          ),
          title: const Text('Job Details'),
        ),
        body: const Center(
          child: Text(
            'No Job Selected',
            style: TextStyle(
              fontSize: 16,
              fontWeight: FontWeight.w600,
              color: AppColors.onSurfaceVariant,
            ),
          ),
        ),
      );
    }

    return Scaffold(
      backgroundColor: AppColors.background,
      body: SafeArea(
        child: JobDetailPanel(
          job: activeJob,
          showBackButton: true,
          onBackPressed: () {
            if (onBackToInbox != null) {
              onBackToInbox!();
            } else {
              Navigator.maybePop(context);
            }
          },
          onJobDismissed: (dismissedJob) {
            ScaffoldMessenger.of(context).showSnackBar(
              SnackBar(
                content: Text('Job "${dismissedJob.title}" hidden from feed'),
              ),
            );
            if (onBackToInbox != null) {
              onBackToInbox!();
            } else {
              Navigator.maybePop(context);
            }
          },
        ),
      ),
    );
  }
}