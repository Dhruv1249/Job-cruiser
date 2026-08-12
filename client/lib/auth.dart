import 'package:flutter/material.dart';
import 'package:flutter/foundation.dart' show kIsWeb;
import 'main.dart' show AppColors, JobCruiserShell;
import 'services/api_service.dart';
import 'package:google_sign_in/google_sign_in.dart';
import 'package:flutter_dotenv/flutter_dotenv.dart';
import 'onboarding.dart';

class AuthScreen extends StatefulWidget {
  const AuthScreen({super.key});

  @override
  State<AuthScreen> createState() => _AuthScreenState();
}

class _AuthScreenState extends State<AuthScreen> {
  final ApiService _apiService = ApiService();

  // Controllers to read the text inputs
  final TextEditingController _emailController = TextEditingController();
  final TextEditingController _passwordController = TextEditingController();

  bool _isLoading = false;

  static String _resolveGoogleClientId() {
    const compileTimeId = String.fromEnvironment('GOOGLE_CLIENT_ID');
    if (compileTimeId.isNotEmpty) {
      return compileTimeId;
    }
    return dotenv.env['GOOGLE_CLIENT_ID'] ?? '';
  }

  final GoogleSignIn _googleSignIn = GoogleSignIn(
    clientId: _resolveGoogleClientId(),
    serverClientId: kIsWeb ? null : _resolveGoogleClientId(),
  );
  @override
  void dispose() {
    // Controllers must be disposed to prevent memory leaks
    _emailController.dispose();
    _passwordController.dispose();
    super.dispose();
  }

  Future<void> _handleLogin() async {
    setState(() {
      _isLoading = true;
    });

    final response = await _apiService.login(
      _emailController.text,
      _passwordController.text,
    );

    if (!mounted) return;

    if (response != null) {
      final hasPreferences = response['has_preferences'] as bool? ?? false;
      if (!hasPreferences) {
        Navigator.of(context).pushReplacement(
          MaterialPageRoute(
            builder: (_) => const OnboardingWizardScreen(),
          ),
        );
      } else {
        Navigator.of(context).pushReplacement(
          MaterialPageRoute(builder: (_) => const JobCruiserShell()),
        );
      }
    } else {
      setState(() {
        _isLoading = false;
      });
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Login failed. Check your credentials.')),
      );
    }
  }

  Future<void> _handleGoogleLogin() async {
    setState(() => _isLoading = true);

    try {
      try {
        await _googleSignIn.signOut();
      } catch (_) {}

      final GoogleSignInAccount? account = await _googleSignIn.signIn();

      if (account == null) {
        if (!mounted) return;
        setState(() => _isLoading = false);
        return;
      }

      final GoogleSignInAuthentication auth = await account.authentication;
      final String? token = (auth.idToken != null && auth.idToken!.isNotEmpty)
          ? auth.idToken
          : auth.accessToken;

      if (token == null || token.isEmpty) {
        throw Exception("No authentication token received");
      }

      final response = await _apiService.googleLogin(token);

      if (!mounted) return;

      if (response != null) {
        final isNewUser = response['is_new_user'] as bool? ?? false;
        final hasPreferences = response['has_preferences'] as bool? ?? false;
        final suggestedName = response['suggested_name'] as String?;

        if (isNewUser || !hasPreferences) {
          Navigator.of(context).pushReplacement(
            MaterialPageRoute(
              builder: (_) => OnboardingWizardScreen(suggestedName: suggestedName),
            ),
          );
        } else {
          Navigator.of(context).pushReplacement(
            MaterialPageRoute(
              builder: (_) => const JobCruiserShell(),
            ),
          );
        }
      } else {
        setState(() => _isLoading = false);
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(
            content: Text("Server rejected login"),
          ),
        );
      }
    } catch (e) {
      if (!mounted) return;
      setState(() => _isLoading = false);
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text("Google Sign-In failed: $e"),
        ),
      );
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: AppColors.surface,
      body: Center(
        child: SingleChildScrollView(
          padding: const EdgeInsets.all(24.0),
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: 400),
            child: Column(
              mainAxisAlignment: MainAxisAlignment.center,
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                const Text(
                  'Job-Cruiser',
                  textAlign: TextAlign.center,
                  style: TextStyle(
                    fontSize: 32,
                    fontWeight: FontWeight.bold,
                    color: AppColors.primary,
                  ),
                ),
                const SizedBox(height: 48),
                TextField(
                  controller: _emailController,
                  keyboardType: TextInputType.emailAddress,
                  decoration: const InputDecoration(
                    labelText: 'Email',
                    border: OutlineInputBorder(),
                  ),
                ),
                const SizedBox(height: 16),
                TextField(
                  controller: _passwordController,
                  obscureText: true,
                  decoration: const InputDecoration(
                    labelText: 'Password',
                    border: OutlineInputBorder(),
                  ),
                ),
                const SizedBox(height: 24),
                SizedBox(
                  height: 48,
                  child: ElevatedButton(
                    onPressed: _isLoading ? null : _handleLogin,
                    style: ElevatedButton.styleFrom(
                      backgroundColor: AppColors.primary,
                      foregroundColor: Colors.white,
                    ),
                    child: _isLoading
                        ? const SizedBox(
                            height: 24,
                            width: 24,
                            child: CircularProgressIndicator(
                              color: Colors.white,
                              strokeWidth: 2,
                            ),
                          )
                        : const Text('Sign In'),
                  ),
                ),
                const SizedBox(height: 24),
                const Row(
                  children: [
                    Expanded(child: Divider()),
                    Padding(
                      padding: EdgeInsets.symmetric(horizontal: 16),
                      child: Text(
                        'OR',
                        style: TextStyle(color: AppColors.outline),
                      ),
                    ),
                    Expanded(child: Divider()),
                  ],
                ),
                const SizedBox(height: 24),

                // Google SSO Button
                SizedBox(
                  height: 48,
                  child: OutlinedButton.icon(
                    onPressed: _isLoading ? null : _handleGoogleLogin,
                    icon: const Icon(
                      Icons.g_mobiledata,
                      size: 28,
                    ), // Built-in G icon
                    label: const Text('Sign in with Google'),
                    style: OutlinedButton.styleFrom(
                      foregroundColor: AppColors.primary,
                      side: const BorderSide(color: AppColors.outlineVariant),
                    ),
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
