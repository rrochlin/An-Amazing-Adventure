#include <functional>
#include <iostream>
#include <vector>

struct TestInfo {
   std::string name;
   std::function<void()> func;
};

inline std::vector<TestInfo> &get_test_registry() {
   static std::vector<TestInfo> registry;
   return registry;
}

inline int register_test_func(std::string name, std::function<void()> func) {
   get_test_registry().push_back({name, func});
   return 0;
}

// Test helper macros
#define TEST(test_name)                                                        \
   void test_name();                                                           \
   static int test_name##registrar =                                           \
       register_test_func(#test_name, test_name);                              \
   void test_name()

#define RUN_TEST(name)                                                         \
   do {                                                                        \
      std::cout << "Running " << #name << "... ";                              \
      try {                                                                    \
         name();                                                               \
         std::cout << "PASSED" << std::endl;                                   \
         tests_passed++;                                                       \
      } catch (const std::exception &e) {                                      \
         std::cout << "FAILED: " << e.what() << std::endl;                     \
         tests_failed++;                                                       \
      } catch (...) {                                                          \
         std::cout << "FAILED: unknown exception" << std::endl;                \
         tests_failed++;                                                       \
      }                                                                        \
   } while (0)

#define ASSERT_TRUE(expr)                                                      \
   do {                                                                        \
      if (!(expr)) {                                                           \
         throw std::runtime_error("Assertion failed: " #expr);                 \
      }                                                                        \
   } while (0)

#define ASSERT_FALSE(expr) ASSERT_TRUE(!(expr))

#define ASSERT_EQ(expected, actual)                                            \
   do {                                                                        \
      if (!((expected) == (actual))) {                                         \
         std::ostringstream oss;                                               \
         oss << "Expected " << (expected) << " but got " << (actual)           \
             << std::endl;                                                     \
         throw std::runtime_error(oss.str());                                  \
      }                                                                        \
   } while (0)

#define ASSERT_NEQ(expected, actual)                                           \
   do {                                                                        \
      if (((expected) == (actual))) {                                          \
         std::ostringstream oss;                                               \
         oss << "Terms were equal: " << (expected) << " and " << (actual)      \
             << std::endl;                                                     \
         throw std::runtime_error(oss.str());                                  \
      }                                                                        \
   } while (0)

#define ASSERT_THROWS(expr, exception_type)                                    \
   do {                                                                        \
      bool caught = false;                                                     \
      try {                                                                    \
         expr;                                                                 \
      } catch (const exception_type &) {                                       \
         caught = true;                                                        \
      }                                                                        \
      if (!caught) {                                                           \
         throw std::runtime_error("Expected exception not thrown: " #expr);    \
      }                                                                        \
   } while (0)
