#pragma once
#include "test_utils.h"
#include <exception>

int main(int argc, char *argv[]) {
   std::string filter = "";
   if (argc > 1) {
      filter = argv[1];
   }

   auto &registry = get_test_registry();
   int passed = 0;
   int failed = 0;

   for (const auto &test : registry) {
      // skip tests that don't match the filter if it was provided
      if (!filter.empty() && test.name.find(filter) == std::string::npos) {
         continue;
      }
      std::cout << "Running " << test.name << "... ";
      try {
         test.func();
         std::cout << "PASSED" << std::endl;
         passed++;
      } catch (const std::exception &e) {
         std::cout << "FAILED: " << e.what() << std::endl;
         failed++;
      }
   }
   std::cout << "Total tests passed: " << passed << std::endl;
   std::cout << "Total tests failed: " << failed << std::endl;
}
