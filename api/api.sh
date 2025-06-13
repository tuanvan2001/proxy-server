#!/bin/bash

# Script để gọi tất cả các API trong dự án proxy-server

# Cấu hình
API_HOST="localhost"
API_PORT="8080"
BASE_URL="http://${API_HOST}:${API_PORT}"
ADMIN_USERNAME="admin"
ADMIN_PASSWORD="Tuandev2001"
TOKEN=""

# Màu sắc cho output
RED="\033[0;31m"
GREEN="\033[0;32m"
YELLOW="\033[0;33m"
BLUE="\033[0;34m"
NC="\033[0m" # No Color

# Hàm hiển thị tiêu đề
print_header() {
    echo -e "\n${BLUE}==== $1 ====${NC}\n"
}

# Hàm hiển thị kết quả
print_result() {
    if [ $1 -eq 0 ]; then
        echo -e "${GREEN}Thành công: $2${NC}"
    else
        echo -e "${RED}Lỗi: $2${NC}"
        echo -e "${RED}Mã lỗi: $1${NC}"
    fi
}

# Hàm đăng nhập và lấy token
login() {
    print_header "Đăng nhập"
    
    echo -e "Đăng nhập với tài khoản: ${YELLOW}$ADMIN_USERNAME${NC}"
    
    response=$(curl -s -X POST "$BASE_URL/login" \
        -H "Content-Type: application/json" \
        -d '{"username":"'"$ADMIN_USERNAME"'", "password":"'"$ADMIN_PASSWORD"'"}' \
        -w "\n%{http_code}")
    
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$ d')
    
    if [ "$http_code" -eq 200 ]; then
        TOKEN=$(echo $body | grep -o '"token":"[^"]*"' | cut -d '"' -f 4)
        expires=$(echo $body | grep -o '"expires":"[^"]*"' | cut -d '"' -f 4)
        print_result 0 "Đăng nhập thành công. Token hết hạn vào: $expires"
        echo -e "Token: ${YELLOW}${TOKEN:0:20}...${NC}"
    else
        print_result $http_code "Đăng nhập thất bại"
        echo -e "Response: $body"
        exit 1
    fi
}

# Hàm lấy danh sách người dùng có phân trang và tìm kiếm
get_users() {
    print_header "Lấy danh sách người dùng"
    
    local page=${1:-1}
    local page_size=${2:-10}
    local search=${3:-""}
    
    local search_param=""
    if [ ! -z "$search" ]; then
        search_param="&search=$search"
    fi
    
    echo -e "Lấy danh sách người dùng: Trang ${YELLOW}$page${NC}, Kích thước trang ${YELLOW}$page_size${NC}, Tìm kiếm: ${YELLOW}$search${NC}"
    
    response=$(curl -s -X GET "$BASE_URL/api/users?page=$page&pageSize=$page_size$search_param" \
        -H "Authorization: Bearer $TOKEN" \
        -w "\n%{http_code}")
    
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$ d')
    
    if [ "$http_code" -eq 200 ]; then
        print_result 0 "Lấy danh sách người dùng thành công"
        echo -e "Response:\n$body"
    else
        print_result $http_code "Lấy danh sách người dùng thất bại"
        echo -e "Response: $body"
    fi
}

# Hàm tạo người dùng mới
create_user() {
    print_header "Tạo người dùng mới"
    
    local username=${1:-"newuser"}
    local password=${2:-"password123"}
    local max_connection=${3:-5}
    
    echo -e "Tạo người dùng mới: ${YELLOW}$username${NC}, Mật khẩu: ${YELLOW}$password${NC}, Kết nối tối đa: ${YELLOW}$max_connection${NC}"
    
    response=$(curl -s -X POST "$BASE_URL/api/users" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $TOKEN" \
        -d '{"username":"'"$username"'", "password":"'"$password"'", "maxConnection":'"$max_connection"'}' \
        -w "\n%{http_code}")
    
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$ d')
    
    if [ "$http_code" -eq 200 ]; then
        print_result 0 "Tạo người dùng thành công"
        echo -e "Response:\n$body"
    else
        print_result $http_code "Tạo người dùng thất bại"
        echo -e "Response: $body"
    fi
}

# Hàm cập nhật thông tin người dùng
update_user() {
    print_header "Cập nhật thông tin người dùng"
    
    local username=${1:-"newuser"}
    local max_connection=${2:-10}
    
    echo -e "Cập nhật người dùng: ${YELLOW}$username${NC}, Kết nối tối đa mới: ${YELLOW}$max_connection${NC}"
    
    response=$(curl -s -X PUT "$BASE_URL/api/users/$username" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $TOKEN" \
        -d '{"maxConnection":'"$max_connection"'}' \
        -w "\n%{http_code}")
    
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$ d')
    
    if [ "$http_code" -eq 200 ]; then
        print_result 0 "Cập nhật người dùng thành công"
        echo -e "Response:\n$body"
    else
        print_result $http_code "Cập nhật người dùng thất bại"
        echo -e "Response: $body"
    fi
}

# Hàm đặt lại mật khẩu người dùng
reset_password() {
    print_header "Đặt lại mật khẩu người dùng"
    
    local username=${1:-"newuser"}
    local new_password=${2:-"newpassword123"}
    
    echo -e "Đặt lại mật khẩu cho người dùng: ${YELLOW}$username${NC}, Mật khẩu mới: ${YELLOW}$new_password${NC}"
    
    response=$(curl -s -X PUT "$BASE_URL/api/users/$username/password" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $TOKEN" \
        -d '{"newPassword":"'"$new_password"'"}' \
        -w "\n%{http_code}")
    
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$ d')
    
    if [ "$http_code" -eq 200 ]; then
        print_result 0 "Đặt lại mật khẩu thành công"
        echo -e "Response:\n$body"
    else
        print_result $http_code "Đặt lại mật khẩu thất bại"
        echo -e "Response: $body"
    fi
}

# Hàm đổi mật khẩu của người dùng đang đăng nhập
change_password() {
    print_header "Đổi mật khẩu của người dùng đang đăng nhập"
    
    local old_password=${1:-"Tuandev2001"}
    local new_password=${2:-"newadminpassword"}
    
    echo -e "Đổi mật khẩu cho người dùng đang đăng nhập: ${YELLOW}$ADMIN_USERNAME${NC}"
    echo -e "Mật khẩu cũ: ${YELLOW}$old_password${NC}, Mật khẩu mới: ${YELLOW}$new_password${NC}"
    
    response=$(curl -s -X PUT "$BASE_URL/api/change-password" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $TOKEN" \
        -d '{"oldPassword":"'"$old_password"'", "newPassword":"'"$new_password"'"}' \
        -w "\n%{http_code}")
    
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$ d')
    
    if [ "$http_code" -eq 200 ]; then
        print_result 0 "Đổi mật khẩu thành công"
        echo -e "Response:\n$body"
    else
        print_result $http_code "Đổi mật khẩu thất bại"
        echo -e "Response: $body"
    fi
}

# Hàm xóa người dùng
delete_user() {
    print_header "Xóa người dùng"
    
    local username=${1:-"newuser"}
    local password=${2:-"testpassword"}
    
    echo -e "Xóa người dùng: ${YELLOW}$username${NC}"
    echo -e "Mật khẩu xác thực: ${YELLOW}$password${NC}"
    
    response=$(curl -s -X DELETE "$BASE_URL/api/users/$username" \
        -H "Authorization: Bearer $TOKEN" \
        -H "Content-Type: application/json" \
        -d '{"password":"'"$password"'"}' \
        -w "\n%{http_code}")
    
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$ d')
    
    if [ "$http_code" -eq 200 ]; then
        print_result 0 "Xóa người dùng thành công"
        echo -e "Response:\n$body"
    else
        print_result $http_code "Xóa người dùng thất bại"
        echo -e "Response: $body"
    fi
}

# Hàm hiển thị menu
show_menu() {
    echo -e "\n${BLUE}===== MENU =====${NC}"
    echo -e "${YELLOW}1.${NC} Đăng nhập"
    echo -e "${YELLOW}2.${NC} Lấy danh sách người dùng"
    echo -e "${YELLOW}3.${NC} Tạo người dùng mới"
    echo -e "${YELLOW}4.${NC} Cập nhật thông tin người dùng"
    echo -e "${YELLOW}5.${NC} Đặt lại mật khẩu người dùng"
    echo -e "${YELLOW}6.${NC} Đổi mật khẩu của người dùng đang đăng nhập"
    echo -e "${YELLOW}7.${NC} Xóa người dùng"
    echo -e "${YELLOW}8.${NC} Chạy tất cả các API"
    echo -e "${YELLOW}0.${NC} Thoát"
    echo -e "${BLUE}================${NC}"
    echo -n "Nhập lựa chọn của bạn: "
    read choice
    
    case $choice in
        1) login ;;
        2) 
            echo -n "Nhập số trang (mặc định: 1): "
            read page
            page=${page:-1}
            
            echo -n "Nhập kích thước trang (mặc định: 10): "
            read page_size
            page_size=${page_size:-10}
            
            echo -n "Nhập từ khóa tìm kiếm (để trống nếu không cần): "
            read search
            
            get_users "$page" "$page_size" "$search"
            ;;
        3)
            echo -n "Nhập tên người dùng: "
            read username
            username=${username:-"newuser"}
            
            echo -n "Nhập mật khẩu: "
            read password
            password=${password:-"password123"}
            
            echo -n "Nhập số kết nối tối đa: "
            read max_connection
            max_connection=${max_connection:-5}
            
            create_user "$username" "$password" "$max_connection"
            ;;
        4)
            echo -n "Nhập tên người dùng cần cập nhật: "
            read username
            username=${username:-"newuser"}
            
            echo -n "Nhập số kết nối tối đa mới: "
            read max_connection
            max_connection=${max_connection:-10}
            
            update_user "$username" "$max_connection"
            ;;
        5)
            echo -n "Nhập tên người dùng cần đặt lại mật khẩu: "
            read username
            username=${username:-"newuser"}
            
            echo -n "Nhập mật khẩu mới: "
            read new_password
            new_password=${new_password:-"newpassword123"}
            
            reset_password "$username" "$new_password"
            ;;
        6)
            echo -n "Nhập mật khẩu cũ: "
            read old_password
            old_password=${old_password:-"Tuandev2001"}
            
            echo -n "Nhập mật khẩu mới: "
            read new_password
            new_password=${new_password:-"newadminpassword"}
            
            change_password "$old_password" "$new_password"
            ;;
        7)
            echo -n "Nhập tên người dùng cần xóa: "
            read username
            username=${username:-"newuser"}
            
            echo -n "Nhập mật khẩu để xác thực: "
            read password
            password=${password:-"testpassword"}
            
            delete_user "$username" "$password"
            ;;
        8)
            run_all
            ;;
        0)
            echo -e "${GREEN}Tạm biệt!${NC}"
            exit 0
            ;;
        *)
            echo -e "${RED}Lựa chọn không hợp lệ!${NC}"
            ;;
    esac
    
    show_menu
}

# Hàm chạy tất cả các API
run_all() {
    print_header "Chạy tất cả các API"
    
    # Đăng nhập
    login
    
    # Tạo người dùng mới
    create_user "testuser" "testpassword" 5
    
    # Lấy danh sách người dùng
    get_users 1 10
    
    # Cập nhật thông tin người dùng
    update_user "testuser" 10
    
    # Đặt lại mật khẩu người dùng
    reset_password "testuser" "newpassword123"
    
    # Đổi mật khẩu của người dùng đang đăng nhập (bỏ qua để không ảnh hưởng đến tài khoản admin)
    # change_password "Tuandev2001" "newadminpassword"
    
    # Xóa người dùng
    delete_user "testuser" "newpassword123"
    
    print_header "Hoàn thành chạy tất cả các API"
}

# Kiểm tra các công cụ cần thiết
check_requirements() {
    if ! command -v curl &> /dev/null; then
        echo -e "${RED}Lỗi: curl không được cài đặt. Vui lòng cài đặt curl và thử lại.${NC}"
        exit 1
    fi
}

# Hàm chính
main() {
    echo -e "${BLUE}===== Script gọi API cho SOCKS5 Proxy Server =====${NC}"
    check_requirements
    
    # Kiểm tra tham số dòng lệnh
    if [ "$1" = "--run-all" ]; then
        login
        run_all
    else
        show_menu
    fi
}

# Chạy hàm chính
main "$@"