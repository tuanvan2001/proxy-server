CREATE DATABASE proxy;
-- Tạo bảng user với các trường yêu cầu
CREATE TABLE IF NOT EXISTS `user` (
  `username` VARCHAR(50) NOT NULL,
  `password` VARCHAR(32) NOT NULL COMMENT 'Mật khẩu được mã hóa bằng MD5',
  `maxConnection` INT NOT NULL DEFAULT 5 COMMENT 'Số lượng kết nối tối đa cho phép',
  `createdAt` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updatedAt` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`username`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Thêm một số dữ liệu mẫu
INSERT INTO `user` (`username`, `password`, `maxConnection`) VALUES
('admin', MD5('Tuandev2001'), 1);