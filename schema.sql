-- Membuat database jika belum ada
CREATE DATABASE IF NOT EXISTS db_wags CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

USE db_wags;

-- Tabel untuk pengguna
CREATE TABLE IF NOT EXISTS `user` (
    `username` VARCHAR(50) NOT NULL,
    `key` VARCHAR(255) NOT NULL, -- Simpan hash password, bukan plain text
    PRIMARY KEY (`username`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Tabel untuk pesan bulk/broadcast
CREATE TABLE IF NOT EXISTS `message_bulk` (
    `id` INT AUTO_INCREMENT,
    `sender` VARCHAR(50) NOT NULL,
    `status` ENUM('PROCESS', 'DONE', 'FAILED') DEFAULT 'PROCESS',
    `dt_store` DATETIME NOT NULL,
    `dt_convert` DATETIME NULL,
    `bulk` JSON NOT NULL,
    PRIMARY KEY (`id`),
    FOREIGN KEY (`sender`) REFERENCES `user`(`username`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Tabel untuk pesan individual
CREATE TABLE IF NOT EXISTS `message` (
    `id` INT AUTO_INCREMENT,
    `sender` VARCHAR(50) NOT NULL,
    `recipient` VARCHAR(20) NOT NULL, -- Nomor telepon, contoh: 628123456789
    `status` ENUM('PENDING', 'SENT', 'FAILED', 'PROCESSING') DEFAULT 'PENDING', -- PROCESSING ditambahkan untuk menandakan sedang dikirim
    `type` VARCHAR(50) NULL, -- Jika berasal dari bulk, simpan message_bulk.id
    `dt_store` DATETIME NOT NULL,
    `dt_queue` DATETIME NOT NULL,
    `dt_send` DATETIME NULL,
    `message` TEXT NOT NULL,
    `external_api_response` TEXT NULL, -- Untuk menyimpan response dari API eksternal
    PRIMARY KEY (`id`),
    FOREIGN KEY (`sender`) REFERENCES `user`(`username`) ON DELETE CASCADE ON UPDATE CASCADE,
    INDEX `idx_status_dt_queue` (`status`, `dt_queue`) -- Index untuk membantu query worker
    -- Jika `type` merujuk ke `message_bulk.id`, bisa ditambahkan FOREIGN KEY constraint
    -- FOREIGN KEY (`type`) REFERENCES `message_bulk`(`id`) ON DELETE SET NULL ON UPDATE CASCADE;
    -- Namun karena `type` adalah VARCHAR untuk menyimpan ID, konversi tipe data perlu diperhatikan jika FK diterapkan.
    -- Untuk kesederhanaan awal, kita biarkan sebagai VARCHAR.
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Contoh data user (password harus di-hash di aplikasi)
-- Ganti 'hashed_password_telkomsel' dengan hasil hash bcrypt atau sejenisnya
INSERT INTO `user` (`username`, `key`) VALUES
('telkomsel', 'W4@4rt47767#');

-- Catatan:
-- 1. `password` di tabel `user` harus disimpan sebagai hash (misalnya menggunakan bcrypt di Go).
-- 2. `dt_queue` di tabel `message` harus di-generate secara hati-hati untuk memastikan keunikan atau setidaknya urutan yang benar.
--    Pertimbangkan untuk menggunakan timestamp dengan presisi tinggi atau mekanisme locking saat mengambil job.
-- 3. `external_api_response` ditambahkan untuk logging.
-- 4. Status 'PROCESSING' ditambahkan di tabel `message` agar worker bisa menandai pesan yang sedang diproses.
-- 5. Index `idx_status_dt_queue` ditambahkan untuk optimasi query pengambilan antrian.
