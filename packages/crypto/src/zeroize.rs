use zeroize::Zeroize;

pub fn zeroize_buffer(buffer: &mut [u8]) {
    buffer.zeroize();
}

pub fn zeroize_buffers(buffers: &mut [&mut [u8]]) {
    for buffer in buffers.iter_mut() {
        buffer.zeroize();
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_zeroize_buffer() {
        let mut key = [1u8; 32];
        zeroize_buffer(&mut key);
        assert_eq!(key, [0u8; 32]);
    }
}