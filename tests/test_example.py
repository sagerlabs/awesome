#!/usr/bin/env python3
"""示例测试文件"""

def test_addition():
    """测试加法"""
    assert 1 + 1 == 2

def test_string_concat():
    """测试字符串拼接"""
    assert "hello" + " " + "world" == "hello world"

def test_list_operations():
    """测试列表操作"""
    my_list = [1, 2, 3]
    my_list.append(4)
    assert my_list == [1, 2, 3, 4]
    assert len(my_list) == 4
