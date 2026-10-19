// a single-line comment
#include <string>
#include <vector>

namespace app {

template <typename T, int N>
class Buffer final {
public:
    explicit Buffer(const T &init) noexcept : value_(init) {}
    virtual ~Buffer() = default;

    constexpr int size() const { return N; }

    auto operator<=>(const Buffer &other) const = default;

private:
    T value_;
    static inline int count_ = 0;
};

struct Point { int x; int y; };

enum class Colour : unsigned char { Red, Green, Blue };

int classify(Colour c)
{
    switch (c) {
    case Colour::Red:
        return 1;
    case Colour::Green:
        return 2;
    default:
        return 0;
    }
}

void demo()
{
    auto million = 1'000'000;
    auto big = 0xdead'beefu;
    auto bits = 0b1010'1010;
    auto oct = 0'755;
    auto pi = 3.141'592f;
    auto sci = 1'0e-3;
    auto wait = 100ms;
    auto angle = 90s;
    auto imag = 2i;

    std::string raw = R"delim(no \escapes "here" at all)delim";
    std::string plain = "with \"escapes\" and \n";
    const char *p = nullptr;

    auto *q = dynamic_cast<Buffer<int, 4> *>(p);
    if (q != nullptr && true) {
        delete q;
    }

    try {
        throw std::runtime_error("boom");
    } catch (const std::exception &e) {
        (void)e;
    }
}

} // namespace app
