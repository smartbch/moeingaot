pragma solidity 0.8.13;

contract babylon {
    uint public result;
    int public fibResult0;
    int public fibResult1;
    function sqrt(uint y) external {
        uint z;
        if (y > 3) {
            z = y;
            uint x = y / 2 + 1;
            while (x < z) {
                z = x;
                x = (y / x + x) / 2;
            }
        } else if (y != 0) {
            z = 1;
        }
        result = z;
        fibResult0 = fib(5);
        fibResult1 = fib(100);
    }

    function fib(int n) public pure returns (int) {
        int t1 = 0;
       	int t2 = 1; 
        for (int i = 1; i <= n; ++i) {
            int nextTerm = t1 + t2;
            t1 = t2;
            t2 = nextTerm;
        }
	return t2;
    }
}
