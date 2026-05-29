import 'package:flutter/material.dart';
import 'main.dart' show AppColors, JobCruiserShell;
import 'services/api_service.dart'; // Adjust path if necessary
import 'package:google_sign_in/google_sign_in.dart';
import 'package:flutter_dotenv/flutter_dotenv.dart';

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

  final GoogleSignIn _googleSignIn = GoogleSignIn(
  serverClientId: dotenv.env['GOOGLE_CLIENT_ID'],
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

    final bool success = await _apiService.login(
      _emailController.text,
      _passwordController.text,
    );

    if (!mounted) return;

    if (success) {
      // Navigate to the main app on success
      Navigator.of(context).pushReplacement(
        MaterialPageRoute(builder: (_) => const JobCruiserShell()),
      );
    } else {
      // Stop loading and show error on failure
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
    debugPrint("Starting Google Sign-In");

    final GoogleSignInAccount? account =
        await _googleSignIn.signIn();

    if (account == null) {
      // User cancelled
      if (!mounted) return;
      setState(() => _isLoading = false);
      return;
    }

    debugPrint("Account: ${account.email}");

    final GoogleSignInAuthentication auth =
        await account.authentication;

    final String? idToken = auth.idToken;

    debugPrint("Token received: ${idToken != null}");

    if (idToken == null) {
      throw Exception("No ID token received");
    }

    final bool success =
        await _apiService.googleLogin(idToken);

    if (!mounted) return;

    if (success) {
      Navigator.of(context).pushReplacement(
        MaterialPageRoute(
          builder: (_) => const JobCruiserShell(),
        ),
      );
    } else {
      setState(() => _isLoading = false);

      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(
          content: Text("Server rejected login"),
        ),
      );
    }
  } catch (e, s) {
    debugPrint("GOOGLE ERROR: $e");
    debugPrint(s.toString());

    if (!mounted) return;

    setState(() => _isLoading = false);

    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: Text("Google Sign-In failed: $e"),
      ),
    );
  }
}
  Future<void> _handleGuestLogin() async {
    // Completely bypass the API and jump straight into the app
    Navigator.of(context).pushReplacement(
      MaterialPageRoute(builder: (_) => const JobCruiserShell()),
    );
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
                const SizedBox(height: 16),

                // Guest Login Button
                TextButton(
                  onPressed: _isLoading ? null : _handleGuestLogin,
                  style: TextButton.styleFrom(
                    foregroundColor: AppColors.secondary,
                  ),
                  child: const Text('Continue as Guest'),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
