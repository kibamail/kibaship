import { customAlphabet } from 'nanoid'

/**
 * NanoId utility for generating short, URL-safe unique identifiers
 */
export class NanoId {
  private static readonly DEFAULT_ALPHABET = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789'
  private static readonly DEFAULT_LENGTH = 10

  /**
   * Generate a NanoId with default alphabet and length
   */
  static generate(length: number = NanoId.DEFAULT_LENGTH): string {
    return customAlphabet(NanoId.DEFAULT_ALPHABET)(length)
  }

  /**
   * Generate a NanoId with custom alphabet and length
   */
  static generateWithAlphabet(alphabet: string, length: number = NanoId.DEFAULT_LENGTH): string {
    return customAlphabet(alphabet)(length)
  }

  /**
   * Generate a NanoId with only numbers
   */
  static generateNumeric(length: number = NanoId.DEFAULT_LENGTH): string {
    return customAlphabet('0123456789')(length)
  }

  /**
   * Generate a NanoId with only lowercase letters
   */
  static generateLowercase(length: number = NanoId.DEFAULT_LENGTH): string {
    return customAlphabet('abcdefghijklmnopqrstuvwxyz')(length)
  }

  /**
   * Generate a NanoId with only uppercase letters
   */
  static generateUppercase(length: number = NanoId.DEFAULT_LENGTH): string {
    return customAlphabet('ABCDEFGHIJKLMNOPQRSTUVWXYZ')(length)
  }
}