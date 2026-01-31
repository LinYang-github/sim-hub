#include <gtest/gtest.h>
#include "simhub/simhub.hpp"

using namespace simhub;

TEST(ResultTest, SuccessCase) {
    auto res = Result<int>::Success(42);
    EXPECT_TRUE(res.ok());
    EXPECT_EQ(res.value, 42);
    EXPECT_EQ(res.code, ErrorCode::Success);
}

TEST(ResultTest, FailureCase) {
    auto res = Result<std::string>::Fail(ErrorCode::NetworkError, "Timeout");
    EXPECT_FALSE(res.ok());
    EXPECT_EQ(res.code, ErrorCode::NetworkError);
    EXPECT_EQ(res.message, "Timeout");
}

TEST(StatusTest, OkCase) {
    Status s = Status::Success(true);
    EXPECT_TRUE(s.ok());
}
