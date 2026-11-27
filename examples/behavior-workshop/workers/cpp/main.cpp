#include <algorithm>
#include <iostream>
#include <stdexcept>
#include <string>

#include <json.hpp>

int main() {
    using nlohmann::json;
    for (std::string line; std::getline(std::cin, line);) {
        try {
            const auto request = json::parse(line);
            if (request.at("version") != 1) {
                throw std::runtime_error("unsupported behavior protocol");
            }
            const auto &actions = request.at("actions");
            const bool launched = std::find(actions.begin(), actions.end(), "launch") != actions.end();
            auto impulses = json::array();
            for (const auto &actor : request.at("actors")) {
                if (actor.at("behavior") != "launch") {
                    throw std::runtime_error("unknown behavior");
                }
                if (launched) {
                    const double strength = actor.at("parameters").value("height_impulse", 5.0);
                    impulses.push_back({{"entity", actor.at("id")}, {"vector", {0, strength, 0}}});
                }
            }
            std::cout << json({{"version", 1}, {"tick", request.at("tick")}, {"impulses", impulses}})
                             .dump() << std::endl;
        } catch (const std::exception &error) {
            std::cerr << error.what() << '\n';
            return 1;
        }
    }
}
