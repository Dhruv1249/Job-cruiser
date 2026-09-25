import 'package:flutter/material.dart';
import 'package:flutter_dotenv/flutter_dotenv.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:flutter_app/models/job.dart';
import 'package:flutter_app/widgets/job_detail_panel.dart';

void main() {
  setUp(() {
    dotenv.loadFromString(envString: 'API_BASE_URL=http://localhost:8080');
  });

  group('MatchedJob Salary Formatting Tests', () {
    test('returns empty string when salary is null or zero (unpaid/not mentioned)', () {
      const jobWithNullSalary = MatchedJob(
        jobId: 'job-1',
        title: 'Software Engineer',
        company: 'Acme Corp',
        location: 'Bengaluru, India',
        isRemote: false,
        source: 'greenhouse',
        url: 'https://example.com/1',
        postedDate: '2026-09-25',
        seniority: 'Mid-Level',
        summary: 'Go engineer needed.',
        matchScore: 85,
        matchReasoning: 'Great skills fit.',
        techStack: ['Go'],
        isMatched: true,
        salaryMin: null,
        salaryMax: null,
      );
      expect(jobWithNullSalary.salaryText, isEmpty);

      final jobWithZeroSalary = jobWithNullSalary.copyWith().copyWith();
      expect(jobWithZeroSalary.salaryText, isEmpty);
    });

    test('formats pure internship with monthly stipend in INR correctly', () {
      const internJob = MatchedJob(
        jobId: 'job-2',
        title: 'Backend Intern',
        company: 'Zomato',
        location: 'Gurugram, India',
        isRemote: false,
        source: 'greenhouse',
        url: 'https://example.com/2',
        postedDate: '2026-09-25',
        seniority: 'Intern',
        summary: '6 months backend internship.',
        matchScore: 90,
        matchReasoning: 'Strong Python skills.',
        techStack: ['Python', 'FastAPI'],
        isMatched: true,
        salaryMin: 25000,
        salaryMax: 25000,
        currency: 'INR',
        salaryPeriod: 'monthly',
        employmentType: 'intern',
      );
      expect(internJob.salaryText, equals('₹25k/mo'));
      expect(internJob.employmentTypeDisplay, equals('Internship'));
    });

    test('formats intern with PPO and stipend correctly', () {
      const internPpoJob = MatchedJob(
        jobId: 'job-3',
        title: 'Software Engineer Intern (PPO)',
        company: 'Swiggy',
        location: 'Bengaluru, India',
        isRemote: false,
        source: 'lever',
        url: 'https://example.com/3',
        postedDate: '2026-09-25',
        seniority: 'Intern',
        summary: 'Internship leading to full-time PPO.',
        matchScore: 95,
        matchReasoning: 'Perfect Go and distributed systems match.',
        techStack: ['Go', 'PostgreSQL'],
        isMatched: true,
        salaryMin: 40000,
        salaryMax: 50000,
        currency: 'INR',
        salaryPeriod: 'stipend',
        employmentType: 'intern_ppo',
      );
      expect(internPpoJob.salaryText, equals('₹40k - ₹50k/stipend'));
      expect(internPpoJob.employmentTypeDisplay, equals('Intern + PPO'));
    });

    test('formats Indian full-time salary in LPA correctly', () {
      const fullTimeInrJob = MatchedJob(
        jobId: 'job-4',
        title: 'Backend Engineer',
        company: 'Razorpay',
        location: 'Bengaluru, India',
        isRemote: true,
        source: 'ashby',
        url: 'https://example.com/4',
        postedDate: '2026-09-25',
        seniority: 'Mid-Level',
        summary: 'Payment infrastructure engineer.',
        matchScore: 92,
        matchReasoning: 'Strong distributed systems background.',
        techStack: ['Go', 'Kafka'],
        isMatched: true,
        salaryMin: 1500000,
        salaryMax: 2000000,
        currency: 'INR',
        salaryPeriod: 'yearly',
        employmentType: 'full_time',
      );
      expect(fullTimeInrJob.salaryText, equals('₹15 - ₹20 LPA'));
      expect(fullTimeInrJob.employmentTypeDisplay, equals('Full-Time'));
    });

    test('formats US full-time salary in USD correctly', () {
      const usJob = MatchedJob(
        jobId: 'job-5',
        title: 'Senior Systems Engineer',
        company: 'Stripe',
        location: 'San Francisco, CA, USA',
        isRemote: true,
        source: 'greenhouse',
        url: 'https://example.com/5',
        postedDate: '2026-09-25',
        seniority: 'Senior',
        summary: 'Core infrastructure team.',
        matchScore: 88,
        matchReasoning: 'Solid Rust and Linux internals experience.',
        techStack: ['Rust', 'Linux'],
        isMatched: true,
        salaryMin: 160000,
        salaryMax: 200000,
        currency: 'USD',
        salaryPeriod: 'yearly',
        employmentType: 'full_time',
      );
      expect(usJob.salaryText, equals('\$160k - \$200k'));
      expect(usJob.employmentTypeDisplay, equals('Full-Time'));
    });

    test('formats contract hourly salary correctly', () {
      const contractJob = MatchedJob(
        jobId: 'job-6',
        title: 'Go Infrastructure Consultant',
        company: 'Contracting Co',
        location: 'Remote (Global)',
        isRemote: true,
        source: 'indeed',
        url: 'https://example.com/6',
        postedDate: '2026-09-25',
        seniority: 'Senior',
        summary: '6-month contracting role.',
        matchScore: 80,
        matchReasoning: 'Good cloud experience.',
        techStack: ['Go', 'Kubernetes'],
        isMatched: true,
        salaryMin: 70,
        salaryMax: 90,
        currency: 'USD',
        salaryPeriod: 'hourly',
        employmentType: 'contract',
      );
      expect(contractJob.salaryText, equals('\$70 - \$90/hr'));
      expect(contractJob.employmentTypeDisplay, equals('Contract'));
    });

    test('infers employment type from title when employmentType field is absent', () {
      const titleInternJob = MatchedJob(
        jobId: 'job-7',
        title: 'Software Development Engineering Intern - Summer 2026',
        company: 'Amazon',
        location: 'Bengaluru, India',
        isRemote: false,
        source: 'greenhouse',
        url: 'https://example.com/7',
        postedDate: '2026-09-25',
        seniority: '',
        summary: 'Summer intern position.',
        matchScore: 85,
        matchReasoning: 'Good Java and DSA background.',
        techStack: ['Java', 'AWS'],
        isMatched: true,
      );
      expect(titleInternJob.employmentTypeDisplay, equals('Internship'));
      expect(titleInternJob.salaryText, isEmpty);
      expect(titleInternJob.isSpecialEmploymentType, isTrue);
    });

    test('distinguishes special employment types from standard full-time', () {
      const fullTimeJob = MatchedJob(
        jobId: 'job-8',
        title: 'Full Stack Engineer',
        company: 'GitLab',
        location: 'Remote',
        isRemote: true,
        source: 'greenhouse',
        url: 'https://example.com/8',
        postedDate: '2026-09-25',
        seniority: 'Mid-Level',
        summary: 'Full-time engineering role.',
        matchScore: 80,
        matchReasoning: 'Fit',
        techStack: ['Ruby'],
        isMatched: true,
        employmentType: 'full_time',
      );
      expect(fullTimeJob.isSpecialEmploymentType, isFalse);

      const contractJob = MatchedJob(
        jobId: 'job-9',
        title: 'DevOps Specialist',
        company: 'Cloud Corp',
        location: 'Remote',
        isRemote: true,
        source: 'ashby',
        url: 'https://example.com/9',
        postedDate: '2026-09-25',
        seniority: 'Senior',
        summary: 'Contracting specialist.',
        matchScore: 80,
        matchReasoning: 'Fit',
        techStack: ['Terraform'],
        isMatched: true,
        employmentType: 'contract',
      );
      expect(contractJob.isSpecialEmploymentType, isTrue);
    });
  });

  group('JobDetailPanel Salary & Employment Type Badge Widget Tests', () {
    testWidgets('renders employment type and salary badges when present', (tester) async {
      await tester.binding.setSurfaceSize(const Size(1024, 768));
      addTearDown(() => tester.binding.setSurfaceSize(null));

      const internPpoJob = MatchedJob(
        jobId: 'job-widget-1',
        title: 'Backend Intern with PPO',
        company: 'Groww',
        location: 'Bengaluru, India',
        isRemote: false,
        source: 'lever',
        url: 'https://example.com/groww-1',
        postedDate: '2026-09-25',
        seniority: 'Intern',
        summary: 'Internship with full-time PPO conversion.',
        matchScore: 94,
        matchReasoning: 'Excellent Go match.',
        techStack: ['Go', 'PostgreSQL'],
        isMatched: true,
        salaryMin: 35000,
        salaryMax: 35000,
        currency: 'INR',
        salaryPeriod: 'monthly',
        employmentType: 'intern_ppo',
      );

      await tester.pumpWidget(
        const MaterialApp(
          home: Scaffold(
            body: JobDetailPanel(job: internPpoJob),
          ),
        ),
      );
      await tester.pumpAndSettle();

      expect(find.text('Intern + PPO'), findsOneWidget);
      expect(find.text('₹35k/mo'), findsOneWidget);
      expect(find.byIcon(Icons.payments_outlined), findsOneWidget);
      expect(find.byIcon(Icons.badge_outlined), findsOneWidget);
    });

    testWidgets('hides salary badge when salary is null or unpaid', (tester) async {
      await tester.binding.setSurfaceSize(const Size(1024, 768));
      addTearDown(() => tester.binding.setSurfaceSize(null));

      const unpaidJob = MatchedJob(
        jobId: 'job-widget-2',
        title: 'Open Source Fellow',
        company: 'Linux Foundation',
        location: 'Remote (Global)',
        isRemote: true,
        source: 'greenhouse',
        url: 'https://example.com/lf-1',
        postedDate: '2026-09-25',
        seniority: 'Junior',
        summary: 'Community fellow program.',
        matchScore: 80,
        matchReasoning: 'Open source enthusiast.',
        techStack: ['C', 'Rust'],
        isMatched: true,
        salaryMin: null,
        salaryMax: null,
        employmentType: 'intern',
      );

      await tester.pumpWidget(
        const MaterialApp(
          home: Scaffold(
            body: JobDetailPanel(job: unpaidJob),
          ),
        ),
      );
      await tester.pumpAndSettle();

      expect(find.text('Internship'), findsOneWidget);
      expect(find.byIcon(Icons.payments_outlined), findsNothing);
    });
  });
}
