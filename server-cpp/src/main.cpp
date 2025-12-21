#include "api_server/src/server.h"
#include <iostream>

int main() {
  Server server("0.0.0.0", "3000");
  server.run();
}
