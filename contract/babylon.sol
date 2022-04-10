pragma solidity 0.8.13;

contract babylon {
    uint public result;
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
    }
}
